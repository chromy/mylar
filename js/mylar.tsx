import {
  type ActionDispatch,
  useState,
  useMemo,
  useReducer,
  useEffect,
  type ReactNode,
} from "react";
import { useLocation } from "wouter";
import { type TileLayout, type DebugInfo, Viewer } from "./viewer.js";
import { z } from "zod";
import { useJsonQuery } from "./query.js";
import { FullScreenDecryptLoader } from "./loader.js";
import { MylarLink } from "./mylar_link.js";
import {
  type Index,
  IndexSchema,
  TILE_SIZE,
  type IndexEntry,
  ResolveCommittishResponseSchema,
  type ResolveCommittishResponse,
  TagListResponseSchema,
  type TagListResponse,
} from "./schemas.js";

const FileLinesSchema = z.string().array();
type FileLines = z.infer<typeof FileLinesSchema>;
import { CommandMenu } from "./command_menu.js";
import { GlassPanel } from "./glass_panel.js";
import { ModalPanel } from "./modal_panel.js";
import {
  type MylarAction,
  settings,
  mylarReducer,
  initialMylarState,
  type MylarState,
  settingsPanelSetting,
  type LayerType,
  getCurrentLayer,
  createChangeLayerAction,
} from "./state.js";

function toTileLayout(index: Index): TileLayout {
  const entries = index.entries ?? [];
  const lastFile = entries[entries.length - 1];
  if (lastFile === undefined) {
    throw new Error("We can't handle zero files");
  }

  const lineCount = lastFile.lineOffset + lastFile.lineCount;
  const tileCount = Math.ceil(lineCount / (TILE_SIZE * TILE_SIZE));

  return {
    lineCount,
    tileCount,
  };
}

function findIndexEntryByLine(
  entries: IndexEntry[],
  lineNumber: number,
): IndexEntry | undefined {
  if (entries.length === 0 || lineNumber < 0) {
    return undefined;
  }

  let left = 0;
  let right = entries.length - 1;
  let result: IndexEntry | undefined = undefined;

  while (left <= right) {
    const mid = Math.floor((left + right) / 2);
    const entry = entries[mid];

    if (entry === undefined) {
      break;
    }

    const entryStart = entry.lineOffset;
    const entryEnd = entry.lineOffset + entry.lineCount - 1;

    if (lineNumber >= entryStart && lineNumber <= entryEnd) {
      return entry;
    } else if (lineNumber < entryStart) {
      right = mid - 1;
    } else {
      left = mid + 1;
    }
  }

  return result;
}

export interface MylarContentProps {
  repo: string;
  commit: string;
  tree: string;
  index: Index;
}

const displayFileContext = settings.addBoolean({
  id: "setting.displayFileContext",
  name: "file context",
});

const MylarContent = ({ repo, commit, tree, index }: MylarContentProps) => {
  const fileCount = (index.entries ?? []).length;
  const lastFile = (index.entries ?? [])[fileCount - 1];
  const lineCount =
    lastFile === undefined ? "-" : lastFile.lineOffset + lastFile.lineCount;

  const [debug, setDebug] = useState<DebugInfo>([]);
  const [state, dispatch] = useReducer(mylarReducer, initialMylarState);
  const [hoveredLineNumber, setHoveredLineNumber] = useState<number>(-1);
  const [, setLocation] = useLocation();

  const layout = useMemo(() => {
    return toTileLayout(index);
  }, [index]);

  const hoveredEntry = useMemo(() => {
    if (hoveredLineNumber < 0) {
      return undefined;
    }
    return findIndexEntryByLine(index.entries ?? [], hoveredLineNumber);
  }, [index.entries, hoveredLineNumber]);

  const hashString = hoveredEntry?.hash
    ? hoveredEntry.hash.map(b => b.toString(16).padStart(2, "0")).join("")
    : "";

  const { data: fileLines } = useJsonQuery(
    {
      path: `/api/compute/lines/${repo}/${hashString}`,
      schema: FileLinesSchema,
      enabled:
        !!hoveredEntry &&
        hashString.length > 0 &&
        displayFileContext.get(state),
    },
    [repo, hashString],
  );

  const contextLines = useMemo(() => {
    if (!fileLines || !hoveredEntry || hoveredLineNumber < 0) {
      return null;
    }

    const fileLineNumber = hoveredLineNumber - hoveredEntry.lineOffset;
    const contextSize = 5;
    const startLine = Math.max(0, fileLineNumber - contextSize);
    const endLine = Math.min(
      fileLines.length - 1,
      fileLineNumber + contextSize,
    );

    return {
      lines: fileLines.slice(startLine, endLine + 1),
      startLineNumber: startLine + 1,
      hoveredFileLineNumber: fileLineNumber + 1,
    };
  }, [fileLines, hoveredEntry, hoveredLineNumber]);

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape" && settingsPanelSetting.get(state)) {
        dispatch(settingsPanelSetting.disable);
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [dispatch, state]);

  return (
    <div className="mylar-content bottom-0 top-0 fixed left-0 right-0">
      <CommandMenu dispatch={dispatch} state={state} />
      <div className="fixed bottom-0 left-0 top-0 right-0">
        <Viewer
          repo={repo}
          commit={commit}
          tree={tree}
          layout={layout}
          setDebug={setDebug}
          dispatch={dispatch}
          state={state}
          setHoveredLineNumber={setHoveredLineNumber}
        />
      </div>
      <GlassPanel area="mylar-layers fixed top-0 left-0">
        <LayersMenu dispatch={dispatch} state={state} />
      </GlassPanel>
      <GlassPanel area="mylar-tags fixed top-0 left-[300px]">
        <TagsMenu repo={repo} />
      </GlassPanel>
      <GlassPanel area="mylar-buttons fixed top-0 right-0">
        <div className="flex gap-2">
          <Button onClick={() => setLocation("/")}>Home</Button>
          <Button onClick={() => dispatch(settingsPanelSetting.enable)}>
            Settings
          </Button>
        </div>
      </GlassPanel>
      <GlassPanel area="fixed top-12 right-0">
        <GesturesHelp />
      </GlassPanel>
      <GlassPanel area="mylar-content-info self-end text-xxs">
        <table className="table-auto w-full">
          <thead></thead>
          <tbody>
            {repo.startsWith("gh:") && (
              <tr>
                <td>GitHub</td>
                <td>
                  <a
                    href={`https://github.com/${repo.slice(3).replace(":", "/")}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-600 hover:text-blue-800 underline"
                  >
                    {repo.slice(3).replace(":", "/")}
                  </a>
                </td>
              </tr>
            )}
            <tr>
              <td>Files</td>
              <td>{fileCount}</td>
            </tr>
            <tr>
              <td>Lines</td>
              <td>{lineCount}</td>
            </tr>
            <tr>
              <td>Commit</td>
              <td className="font-mono">
                {repo.startsWith("gh:") ? (
                  <a
                    href={`https://github.com/${repo.slice(3).replace(":", "/")}/commit/${commit}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-600 hover:text-blue-800 underline"
                  >
                    {commit.slice(0, 6)}
                  </a>
                ) : (
                  commit.slice(0, 6)
                )}
              </td>
            </tr>
            <tr>
              <td>Tree</td>
              <td className="font-mono">{tree.slice(0, 6)}</td>
            </tr>
            {debug.map(kv => (
              <tr>
                <td>{kv[0]}</td>
                <td>
                  <pre>{kv[1]}</pre>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </GlassPanel>

      <GlassPanel area="mylar-content-line self-end">
        <div className="font-mono text-xxs space-y-1">
          {hoveredEntry &&
            contextLines &&
            displayFileContext.get(state) &&
            contextLines.lines.map((line, index) => {
              const lineNum = contextLines.startLineNumber + index;
              const isHovered = lineNum === contextLines.hoveredFileLineNumber;
              return (
                <div
                  key={lineNum}
                  className={`flex ${isHovered ? "bg-yellow-500/20" : ""}`}
                >
                  <span className="text-gray-600 text-right w-8 mr-2 select-none">
                    {lineNum}
                  </span>
                  <span className="whitespace-pre text-gray-700 overflow-hidden text-ellipsis block max-w-96">
                    {line || " "}
                  </span>
                </div>
              );
            })}
          <div className="flex items-center justify-between text-xxs mt-2">
            <div className="text-gray-600">
              {hoveredEntry ? (
                <span>
                  {hoveredEntry.path}{" "}
                  {contextLines && (
                    <span>
                      (lines {contextLines.startLineNumber}-
                      {contextLines.startLineNumber +
                        contextLines.lines.length -
                        1}
                      )
                    </span>
                  )}
                </span>
              ) : (
                <span>No file selected</span>
              )}
            </div>
            <button
              onClick={() => {
                if (displayFileContext.get(state)) {
                  dispatch(displayFileContext.disable);
                } else {
                  dispatch(displayFileContext.enable);
                }
              }}
              className="ml-4 px-2 py-1 text-xxs bg-white/10 hover:bg-white/20 rounded border border-black/10 transition-colors"
              title={
                displayFileContext.get(state)
                  ? "Hide file context"
                  : "Show file context"
              }
            >
              {displayFileContext.get(state) ? "Hide Context" : "Show Context"}
            </button>
          </div>
        </div>
      </GlassPanel>

      <SettingsPanel dispatch={dispatch} state={state} />
    </div>
  );
};

//      {hoveredEntry && (
//        <GlassPanel area="mylar-content-line self-end grid grid-cols-[auto_1fr] gap-x-5">
//          <div className="text-left pb-2">Line {hoveredLineNumber}</div>
//          <div></div>
//          <div>File</div>
//          <div className="font-mono text-sm">{hoveredEntry.path}</div>
//          <div>Line Range</div>
//          <div>{hoveredEntry.lineOffset} - {hoveredEntry.lineOffset + hoveredEntry.lineCount - 1}</div>
//          <div>File Lines</div>
//          <div>{hoveredEntry.lineCount}</div>
//          <div>File Line</div>
//          <div>{hoveredLineNumber - hoveredEntry.lineOffset + 1}</div>
//        </GlassPanel>
//      )}

const MylarLoading = () => {
  return <FullScreenDecryptLoader />;
};

export interface MylarProps {
  repo: string;
  committish: string;
}

export const Mylar = ({ repo, committish }: MylarProps) => {
  const {
    data: repoData,
    isLoading: repoLoading,
    isError: repoError,
    error: repoErrorMsg,
  } = useJsonQuery({
    path: `/api/resolve/${repo}/${committish}`,
    schema: ResolveCommittishResponseSchema,
  });

  const {
    data: indexData,
    isLoading: indexLoading,
    isError: indexError,
    error: indexErrorMsg,
  } = useJsonQuery({
    path: `/api/repo/${repo}/${repoData?.commit}/index`,
    schema: IndexSchema,
    enabled: !!repoData?.commit,
  });

  if (indexError) {
    throw indexErrorMsg;
  }

  if (repoError) {
    throw repoErrorMsg;
  }

  const isLoading = indexLoading || repoLoading;
  const hasAllData = indexData && repoData;

  return (
    <>
      {isLoading && <FullScreenDecryptLoader />}
      {hasAllData && (
        <MylarContent
          repo={repo}
          commit={repoData.commit}
          tree={repoData.tree}
          index={indexData}
        />
      )}
    </>
  );
};

interface LayersMenuProps {
  dispatch: ActionDispatch<[action: MylarAction]>;
  state: MylarState;
}

const LAYER_OPTIONS: LayerType[] = [
  { kind: "offset", composite: "direct" },
  { kind: "length", composite: "direct" },
  { kind: "fileHash", composite: "hash" },
  { kind: "fileExtension", composite: "hash" },
];

const LAYER_LABELS: Record<string, string> = {
  offset: "Line Offset",
  length: "Line Length",
  fileHash: "File Hash",
  fileExtension: "File Type",
};

const LayersMenu = ({ dispatch, state }: LayersMenuProps) => {
  const currentLayer = getCurrentLayer(state);

  return (
    <div className="space-y-1">
      <div className="text-xs font-medium mb-2">Layers</div>
      <div className="space-y-1">
        {LAYER_OPTIONS.map(layer => (
          <button
            key={`${layer.kind}-${layer.composite}`}
            onClick={() => dispatch(createChangeLayerAction(layer))}
            className={`block w-full text-left px-2 py-1 text-xs rounded-xs transition-colors ${
              currentLayer.kind === layer.kind &&
              currentLayer.composite === layer.composite
                ? "bg-blue-500/20 text-blue-700"
                : "hover:bg-white/10"
            }`}
          >
            {LAYER_LABELS[layer.kind]}
          </button>
        ))}
      </div>
    </div>
  );
};

const GesturesHelp = () => {
  return (
    <div className="space-y-2 text-xs">
      <div className="text-xs font-medium mb-2">Gestures</div>
      <div className="space-y-1">
        <div className="flex items-center justify-between">
          <span>Scroll to pan</span>
        </div>
        <div className="flex items-center justify-between">
          <span>Pinch to zoom</span>
        </div>
        <div className="flex items-center justify-between">
          <span>Zoom</span>
          <div className="flex items-center gap-1">
            <kbd className="gesture-key">⌘</kbd>
            <span className="text-xs">+</span>
            <span className="text-xs">scroll</span>
          </div>
        </div>
        <div className="flex items-center justify-between">
          <span>Command menu</span>
          <div className="flex items-center gap-1">
            <kbd className="gesture-key">⌘</kbd>
            <span className="text-xs">+</span>
            <kbd className="gesture-key">K</kbd>
          </div>
        </div>
      </div>
    </div>
  );
};

interface TagsMenuProps {
  repo: string;
}

const TagsMenu = ({ repo }: TagsMenuProps) => {
  const { data: tagsData } = useJsonQuery({
    path: `/api/tags/${repo}`,
    schema: TagListResponseSchema,
  });

  const tags = tagsData?.tags ?? [];

  if (tags.length === 0) {
    return null;
  }

  return (
    <div className="space-y-1">
      <div className="text-xs font-medium mb-2">Tags</div>
      <div className="space-y-1 max-h-32 overflow-y-auto">
        {tags.map(tag => (
          <MylarLink
            key={tag.tag}
            href={`/app/repo/${repo}/${tag.tag}`}
            className="block w-full text-left px-2 py-1 text-xs rounded-xs hover:bg-white/10 transition-colors text-inherit hover:text-inherit no-underline hover:no-underline"
          >
            <div className="font-medium">{tag.tag}</div>
            <div className="text-xxs text-gray-600 font-mono">
              {tag.commit.slice(0, 6)}
            </div>
          </MylarLink>
        ))}
      </div>
    </div>
  );
};

interface SettingsPanelProps {
  dispatch: ActionDispatch<[action: MylarAction]>;
  state: MylarState;
}

const SettingsPanel = ({ dispatch, state }: SettingsPanelProps) => {
  return (
    <ModalPanel
      isOpen={settingsPanelSetting.get(state)}
      onClose={() => dispatch(settingsPanelSetting.disable)}
      title="Settings"
    >
      <div className="space-y-4">
        {settings.items.map(s => (
          <div>
            <label className="block mb-2">{s.name}</label>
            <p>Value: {s.get(state) ? "True" : "False"}</p>
            <Button onClick={() => dispatch(s.enable)}>Enable</Button>
            <Button onClick={() => dispatch(s.disable)}>Disable</Button>
          </div>
        ))}
      </div>
    </ModalPanel>
  );
};

interface ButtonProps {
  onClick: () => void;
  children: ReactNode;
}

const Button = ({ onClick, children }: ButtonProps) => (
  <button
    className="px-3 py-1 rounded-xs hover:bg-white/10 transition-colors border border-solid rounded-xs border-black/5"
    onClick={onClick}
  >
    {children}
  </button>
);

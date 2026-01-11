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
import { LAYER_LABELS, LAYER_OPTIONS, type LayerType } from "./layers.js";
import { CANCELLED } from "./store.js";

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
  gesturesPanelSetting,
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
  const [hoveredOutline, setHoveredOutline] = useState<Uint8Array|undefined>(undefined);

  const layout = useMemo(() => {
    return toTileLayout(index);
  }, [index]);

  const hoveredEntry = useMemo(() => {
    if (hoveredLineNumber < 0) {
      return undefined;
    }
    return findIndexEntryByLine(index.entries ?? [], hoveredLineNumber);
  }, [index.entries, hoveredLineNumber]);

  useEffect(() => {
    if (hoveredEntry === undefined) {
      setHoveredOutline(undefined);
      return;
    }
    const hash = hoveredEntry.hash.map(b => b.toString(16).padStart(2, "0")).join("")
    const url = `/api/commit/fileQuadtree/${repo}/${commit}/${hash}`;

    const controller = new AbortController();
    const signal = controller.signal;

    fetch(url, {signal})
      .then(response => {
        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        return response.json();
      })
      .then(data => {
        const base64String = data as string;
        const binaryString = atob(base64String);
        const bytes = new Uint8Array(binaryString.length);
        for (let i = 0; i < binaryString.length; i++) {
          bytes[i] = binaryString.charCodeAt(i);
        }
        if (!signal.aborted) {
          setHoveredOutline(bytes);
        }
      })
      .catch(error => {
        if (error !== CANCELLED) {
          console.error("Error fetching quadtree:", error);
        }
      });

      return () => {
        controller.abort(CANCELLED);
      };
  }, [setHoveredOutline, repo, hoveredEntry, commit]);

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

  //console.log(hoveredLineNumber, hoveredOutline, hoveredEntry);

  return (
    <div className="mylar bottom-0 top-0 fixed left-0 right-0">
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
          hoveredOutline={hoveredOutline}
          setHoveredLineNumber={setHoveredLineNumber}
        />
      </div>
      <div className="mylar-layers">
        <GlassPanel>
          <LayersMenu dispatch={dispatch} state={state} />
        </GlassPanel>
        <GlassPanel className="">
          <div className="flex items-center justify-between mb-1">
            <div className="font-medium">Shader</div>
            <button
              className="px-1.5 py-0.5 text-xxs bg-white/10 hover:bg-white/20 rounded-xs border border-black/10 transition-colors"
              onClick={() => {
                // TODO: Implement shader editing
                console.log("Edit shader clicked");
              }}
            >
              Edit
            </button>
          </div>
          <div className="font-mono text-xs break-all">
            {getCurrentLayer(state)
              .composite.split("|")
              .map(p => (
                <div>{p}</div>
              ))}
          </div>
        </GlassPanel>
      </div>
      <TagsMenuWithData repo={repo} />
      <div className="mylar-menu">
        <GlassPanel>
          <div className="flex gap-2">
            <Button onClick={() => setLocation("/")}>Home</Button>
            <Button onClick={() => dispatch(settingsPanelSetting.enable)}>
              Settings
            </Button>
          </div>
        </GlassPanel>
        {gesturesPanelSetting.get(state) && (
          <GlassPanel>
            <GesturesHelp />
          </GlassPanel>
        )}
      </div>
      <GlassPanel className="mylar-debug self-end text-xxs">
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

      <GlassPanel className="mylar-context self-end">
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
                  <span className="whitespace-pre text-gray-700 overflow-hidden text-ellipsis block">
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
              className="ml-4 px-2 py-1 text-xxs bg-white/10 hover:bg-white/20 rounded-xs border border-black/10 transition-colors"
              title={
                displayFileContext.get(state)
                  ? "Hide file context"
                  : "Show file context"
              }
            >
              {displayFileContext.get(state)
                ? "Hide file context"
                : "Show file context"}
            </button>
          </div>
        </div>
      </GlassPanel>

      <SettingsPanel dispatch={dispatch} state={state} />
    </div>
  );
};

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
          <span>Pan</span>
          <span>Scroll</span>
        </div>
        <div className="flex items-center justify-between">
          <span>Pan</span>
          <div className="flex items-center gap-1">
            <kbd className="gesture-key">⌘</kbd>
            <span className="text-xs">click + drag</span>
          </div>
        </div>
        <div className="flex items-center justify-between">
          <span>Pan</span>
          <span className="text-xs">Middle click + drag</span>
        </div>
        <div className="flex items-center justify-between">
          <span>Zoom</span>
          <span>Pinch</span>
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
          <span>Commands</span>
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

interface TagsMenuWithDataProps {
  repo: string;
}

const TagsMenuWithData = ({ repo }: TagsMenuWithDataProps) => {
  const { data: tagsData } = useJsonQuery({
    path: `/api/tags/${repo}`,
    schema: TagListResponseSchema,
  });

  const tags = tagsData?.tags ?? [];

  if (tags.length === 0) {
    return null;
  }

  return (
    <GlassPanel className="mylar-tags max-h-32 flex flex-col">
      <div className="text-xs font-medium mb-2">Tags</div>
      <div className="space-y-1 overflow-y-auto">
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
    </GlassPanel>
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

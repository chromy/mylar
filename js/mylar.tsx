import {
  type ActionDispatch,
  useState,
  useMemo,
  useReducer,
  useEffect,
  type ReactNode,
} from "react";
import { type TileLayout, type DebugInfo, Viewer } from "./viewer.js";
import { z } from "zod";
import { useJsonQuery } from "./query.js";
import { FullScreenDecryptLoader } from "./loader.js";
import {
  type Index,
  IndexSchema,
  TILE_SIZE,
  type IndexEntry,
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

interface IndexPanelProps {
  repo: string;
  committish: string;
}

const IndexPanel = ({ repo, committish }: IndexPanelProps) => {
  const { data, isLoading, isError, error } = useJsonQuery({
    path: `/api/repo/${repo}/${committish}/index/`,
    schema: IndexSchema,
  });

  if (isError) {
    throw error;
  }

  return (
    <div>
      {isLoading && <FullScreenDecryptLoader />}
      <ul>
        {data &&
          (data.entries ?? []).map(e => (
            <li>
              {e.path} {e.lineOffset} {e.lineCount}
            </li>
          ))}
      </ul>
    </div>
  );
};

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
  committish: string;
  index: Index;
}

const displayFileContext = settings.addBoolean({
  id: "setting.displayFileContext",
  name: "file context",
});

const MylarContent = ({ repo, committish, index }: MylarContentProps) => {
  const fileCount = (index.entries ?? []).length;
  const lastFile = (index.entries ?? [])[fileCount - 1];
  const lineCount =
    lastFile === undefined ? "-" : lastFile.lineOffset + lastFile.lineCount;

  const [debug, setDebug] = useState<DebugInfo>([]);
  const [state, dispatch] = useReducer(mylarReducer, initialMylarState);
  const [hoveredLineNumber, setHoveredLineNumber] = useState<number>(-1);

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
      enabled: !!hoveredEntry && hashString.length > 0,
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
          committish={committish}
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
      <GlassPanel area="mylar-buttons fixed top-0 right-0">
        <div className="flex gap-2">
          <Button onClick={() => dispatch(settingsPanelSetting.enable)}>
            Settings
          </Button>
        </div>
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
              <td>Ref</td>
              <td>{committish}</td>
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

      {hoveredEntry && (
        <GlassPanel area="mylar-content-line self-end">
          <div className="font-mono text-xxs space-y-1">
            {contextLines &&
              displayFileContext.get(state) &&
              contextLines.lines.map((line, index) => {
                const lineNum = contextLines.startLineNumber + index;
                const isHovered =
                  lineNum === contextLines.hoveredFileLineNumber;
                return (
                  <div
                    key={lineNum}
                    className={`flex ${isHovered ? "bg-yellow-500/20" : ""}`}
                  >
                    <span className="text-gray-500 text-right w-8 mr-2 select-none">
                      {lineNum}
                    </span>
                    <span className="whitespace-pre">{line || " "}</span>
                  </div>
                );
              })}
            <div className="text-xxs text-gray-400 mt-2">
              {hoveredEntry.path}{" "}
              {contextLines && (
                <span>
                  (lines {contextLines.startLineNumber}-
                  {contextLines.startLineNumber + contextLines.lines.length - 1}
                  )
                </span>
              )}
            </div>
          </div>
        </GlassPanel>
      )}

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
  const { data, isLoading, isError, error } = useJsonQuery({
    path: `/api/repo/${repo}/${committish}/index`,
    schema: IndexSchema,
  });

  if (isError) {
    throw error;
  }

  return (
    <>
      {isLoading && <FullScreenDecryptLoader />}
      {data && (
        <MylarContent repo={repo} committish={committish} index={data} />
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
  { kind: "fileHash", composite: "direct" },
  { kind: "fileExtension", composite: "direct" },
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

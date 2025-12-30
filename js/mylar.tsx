import { useState, useMemo, useReducer } from "react";
import { type TileLayout, type DebugInfo, Viewer } from "./viewer.js";
import { z } from "zod";
import { useJsonQuery } from "./query.js";
import { DecryptLoader } from "./loader.js";
import { type Index, IndexSchema, TILE_SIZE } from "./schemas.js";
import { CommandMenu } from "./command_menu.js";
import { GlassPanel } from "./glass_panel.js";
import { ModalPanel } from "./modal_panel.js";
import { mylarReducer, initialMylarState, type MylarState } from "./state.js"

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
      {isLoading && <DecryptLoader />}
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

export interface MylarContentProps {
  repo: string;
  committish: string;
  index: Index;
}

const MylarContent = ({ repo, committish, index }: MylarContentProps) => {
  const fileCount = (index.entries ?? []).length;
  const lastFile = (index.entries ?? [])[fileCount - 1];
  const lineCount =
    lastFile === undefined ? "-" : lastFile.lineOffset + lastFile.lineCount;

  const [debug, setDebug] = useState<DebugInfo>([]);
  const [state, dispatch] = useReducer(mylarReducer, initialMylarState);

  const layout = useMemo(() => {
    return toTileLayout(index);
  }, [index]);

  return (
    <div className="mylar-content bottom-0 top-0 fixed left-0 right-0">
      <CommandMenu dispatch={dispatch} state={state} />
      <div className="fixed bottom-0 left-0 top-0 right-0">
        <Viewer
          repo={repo}
          committish={committish}
          layout={layout}
          setDebug={setDebug}
        />
      </div>
      <GlassPanel area="mylar-buttons fixed top-0 right-0">
        <div className="flex gap-2">
          <button
            className="px-3 py-1 rounded hover:bg-white/10 transition-colors"
            onClick={() => dispatch({ type: 'TOGGLE_SETTINGS' })}
          >
            Settings
          </button>
          <button
            className="px-3 py-1 rounded hover:bg-white/10 transition-colors"
            onClick={() => dispatch({ type: 'TOGGLE_HELP' })}
          >
            Help
          </button>
        </div>
      </GlassPanel>
      <GlassPanel>
        <table className="table-auto w-full">
          <thead></thead>
          <tbody>
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

      <ModalPanel
        isOpen={state.showSettings}
        onClose={() => dispatch({ type: 'CLOSE_ALL_PANELS' })}
        title="Settings"
      >
        <div className="space-y-4">
          <div>
            <label className="block mb-2">Theme</label>
            <select className="w-full p-2 rounded bg-white/10 border border-white/20">
              <option>Light</option>
              <option>Dark</option>
              <option>Auto</option>
            </select>
          </div>
          <div>
            <label className="block mb-2">Zoom Level</label>
            <input
              type="range"
              min="50"
              max="200"
              defaultValue="100"
              className="w-full"
            />
          </div>
        </div>
      </ModalPanel>

      <ModalPanel
        isOpen={state.showHelp}
        onClose={() => dispatch({ type: 'CLOSE_ALL_PANELS' })}
        title="Help"
      >
        <div className="space-y-4">
          <div>
            <h3 className="font-medium mb-2">Navigation</h3>
            <ul className="space-y-1 text-sm">
              <li>• Pan: Click and drag</li>
              <li>• Zoom: Mouse wheel</li>
              <li>• Reset: Double click</li>
            </ul>
          </div>
          <div>
            <h3 className="font-medium mb-2">Keyboard Shortcuts</h3>
            <ul className="space-y-1 text-sm">
              <li>• <kbd className="bg-white/20 px-1 rounded">?</kbd> - Show help</li>
              <li>• <kbd className="bg-white/20 px-1 rounded">Esc</kbd> - Close panels</li>
            </ul>
          </div>
        </div>
      </ModalPanel>
    </div>
  );
};

const MylarLoading = () => {
  return <DecryptLoader />;
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
      {isLoading && <DecryptLoader />}
      {data && (
        <MylarContent repo={repo} committish={committish} index={data} />
      )}
    </>
  );
};

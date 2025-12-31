import {
  type ActionDispatch,
  useState,
  useMemo,
  useReducer,
  type ReactNode,
} from "react";
import { type TileLayout, type DebugInfo, Viewer } from "./viewer.js";
import { z } from "zod";
import { useJsonQuery } from "./query.js";
import { FullScreenDecryptLoader } from "./loader.js";
import { type Index, IndexSchema, TILE_SIZE } from "./schemas.js";
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
          dispatch={dispatch}
          state={state}
        />
      </div>
      <GlassPanel area="mylar-buttons fixed top-0 right-0">
        <div className="flex gap-2">
          <Button onClick={() => dispatch(settingsPanelSetting.enable)}>
            Settings
          </Button>
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

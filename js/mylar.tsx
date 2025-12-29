import { useState, useMemo } from "react";
import { type TileLayout, type DebugInfo, Viewer } from "./viewer.js";
import { z } from "zod";
import { useJsonQuery } from "./query.js";
import { DecryptLoader } from "./loader.js";
import { type Index, IndexSchema, TILE_SIZE } from "./schemas.js";

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
  const tileCount = Math.ceil(lineCount / (TILE_SIZE*TILE_SIZE));

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

  const layout = useMemo(() => {
    return toTileLayout(index);
  }, [index]);

  return (
    <div className="mylar-content bottom-0 top-0 fixed left-0 right-0">
      <div className="fixed bottom-0 left-0 top-0 right-0">
        <Viewer repo={repo} committish={committish} layout={layout} setDebug={setDebug}/>
      </div>
      <div className="mylar-content-info backdrop-blur-sm z-1 border border-solid rounded-xs border-black/5 m-1 p-2">
        <table className="table-auto w-full text-zinc-950/80 text-xs">
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
            {debug.map(kv => (<tr><td>{kv[0]}</td><td><pre>{kv[1]}</pre></td></tr>))}
          </tbody>
        </table>
      </div>
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

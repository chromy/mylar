import { createRoot } from "react-dom/client";
import { Link, Route, Switch, useParams } from "wouter";
import { Viewer } from "./viewer.js";
import { z } from "zod";
import { DecryptLoader } from "./DecryptLoader.js";
import { useJsonQuery } from "./query.js";

async function fetchJsonQueryFn(signal: AbortSignal): Promise<unknown> {
  const response = await fetch("/api/fs/get", { signal });
  if (!response.ok) {
    throw new Error("Failed to fetch");
  }
  return await response.json();
}

//const FileMetadataReponseSchema = z.object({
//  path: z.string(),
//  name: z.string(),
//  size: z.number(),
//  isDir: z.boolean(),
//  children: z.optional(z.array(z.string())),
//});

const RepoInfoSchema = z.object({
  name: z.string(),
});

const RepoListResponseSchema = z.object({
  repos: z.array(RepoInfoSchema),
});

const IndexStatusResponseSchema = z.object({
  message: z.string(),
  fileCount: z.number(),
});

const Home = () => {
  const { data, isLoading, isError, error } = useJsonQuery({
    path: "/api/repo",
    schema: RepoListResponseSchema,
  });

  return (
    <div className="grid place-content-center">
      <div className="max-w-xl mx-auto w-100 my-4 p-3 border rounded-xs border-black shadow-sm">
        {isLoading && <DecryptLoader />}

        {data &&
          data.repos.map(r => (
            <Link href={`/app/repo/${r.name}/HEAD`}>{r.name}</Link>
          ))}
      </div>
    </div>
  );
};

const IndexEntry = z.object({
  path: z.string(),
  lineOffset: z.number(),
  lineCount: z.number(),
});

const IndexResponseSchema = z.object({
  entries: z.array(IndexEntry),
});

interface IndexPanelProps {
  repo: string;
  committish: string;
}

const IndexPanel = ({ repo, committish }: IndexPanelProps) => {
  const { data, isLoading, isError, error } = useJsonQuery({
    path: `/api/repo/${repo}/${committish}/index`,
    schema: IndexResponseSchema,
  });

  if (isError) {
    throw new Error(error);
  }

  return (
    <div>
      {isLoading && <DecryptLoader />}
      <ul>
        {data &&
          data.entries.map(e => (
            <li>
              {e.path} {e.lineOffset} {e.lineCount}
            </li>
          ))}
      </ul>
    </div>
  );
};

const Repo = () => {
  const params = useParams();
  const repo = params.repo || "";
  const committish = params.committish || "";

  return (
    <div className="grid">
      <IndexPanel repo={repo} committish={committish} />
      <div className="absolute bottom-0 left-0 top-0 right-0">
        <Viewer repo={repo} committish={committish} />
      </div>
    </div>
  );
};

const App = () => (
  <>
    <Switch>
      <Route path="/" component={Home} />
      <Route path="/app/repo/:repo/:committish" component={Repo} />
      <Route>404: No such page!</Route>
    </Switch>
  </>
);

export function main() {
  const dom = document.querySelector("main")!;
  const root = createRoot(dom);
  root.render(<App />);
}

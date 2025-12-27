import { createRoot } from "react-dom/client";
import { Link, Route, Switch, useParams } from "wouter";
import { z } from "zod";
import { DecryptLoader } from "./loader.js";
import { useJsonQuery } from "./query.js";
import { Mylar } from "./mylar.js";

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

const HomePage = () => {
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

const RepoPage = () => {
  const params = useParams();
  const repo = params.repo || "";
  const committish = params.committish || "";

  return <Mylar repo={repo} committish={committish} />;
};

const App = () => (
  <>
    <Switch>
      <Route path="/" component={HomePage} />
      <Route path="/app/repo/:repo/:committish" component={RepoPage} />
      <Route>404: No such page!</Route>
    </Switch>
  </>
);

export function main() {
  const dom = document.querySelector("main")!;
  const root = createRoot(dom);
  root.render(<App />);
}

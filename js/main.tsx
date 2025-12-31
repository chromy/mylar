import { createRoot } from "react-dom/client";
import { Route, Switch, useParams } from "wouter";
import { z } from "zod";
import { FullScreenDecryptLoader } from "./loader.js";
import { useJsonQuery } from "./query.js";
import { Mylar } from "./mylar.js";
import {RepoListResponseSchema} from "./schemas.js"
import { MylarLink } from "./mylar_link.js";

async function fetchJsonQueryFn(signal: AbortSignal): Promise<unknown> {
  const response = await fetch("/api/fs/get", { signal });
  if (!response.ok) {
    throw new Error("Failed to fetch");
  }
  return await response.json();
}

const HomePage = () => {
  const { data, isLoading, isError, error } = useJsonQuery({
    path: "/api/repo",
    schema: RepoListResponseSchema,
  });

  return (
    <div className="grid place-content-center">
      <div className="max-w-xl mx-auto w-100 my-4 p-3 border rounded-xs border-black shadow-sm flex flex-col">
        {isLoading && <FullScreenDecryptLoader />}

        {data?.repos &&
          (
            data.repos.map(r => (
              <MylarLink href={`/app/repo/${r.id}/HEAD`}>{`${r.owner ?? "?"}/${r.name ?? "?"} (${r.id})`}</MylarLink>
            )
                          )
          )}
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

import { createRoot } from "react-dom/client";
import { Route, Switch, useParams, useLocation } from "wouter";
import { z } from "zod";
import { useState } from "react";
import * as Sentry from "@sentry/react";
import { FullScreenDecryptLoader } from "./loader.js";
import { useJsonQuery } from "./query.js";
import { Mylar } from "./mylar.js";
import { RepoListResponseSchema } from "./schemas.js";
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

  const [, setLocation] = useLocation();
  const [githubRepo, setGithubRepo] = useState("");

  const handleGithubRepoSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (githubRepo.trim()) {
      const formatted = githubRepo.trim().replace("/", ":");
      setLocation(`/app/repo/gh:${formatted}/HEAD`);
    }
  };

  return (
    <div className="grid place-content-center">
      <div className="max-w-xl mx-auto w-100 my-4 p-3 border rounded-xs border-black shadow-sm flex flex-col">
        <form onSubmit={handleGithubRepoSubmit} className="mb-4">
          <input
            type="text"
            value={githubRepo}
            onChange={e => setGithubRepo(e.target.value)}
            placeholder="Enter GitHub repo (e.g., google/perfetto)"
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </form>

        {isLoading && <FullScreenDecryptLoader />}

        {data?.repos &&
          data.repos.map(r => (
            <MylarLink
              key={r.id}
              href={`/app/repo/${r.id}/HEAD`}
            >{`${r.owner ?? "?"}/${r.name ?? "?"} (${r.id})`}</MylarLink>
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

function initSentry() {
  const dsn = (window as any).__SENTRY_DSN__;
  if (!dsn) {
    console.log("Sentry DSN not available, skipping initialization");
    return;
  }

  Sentry.init({
    dsn,
    environment: (window as any).__ENVIRONMENT__ || "development",
    integrations: [
      Sentry.browserTracingIntegration(),
      Sentry.replayIntegration(),
    ],
    tracesSampleRate: 1.0,
    replaysSessionSampleRate: 0.1,
    replaysOnErrorSampleRate: 1.0,
  });
}

export function main() {
  initSentry();

  const dom = document.querySelector("main")!;
  const root = createRoot(dom);
  root.render(<App />);
}

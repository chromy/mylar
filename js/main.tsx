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
    <div className="min-h-screen bg-gradient-to-br from-slate-50 to-slate-100">
      <header className="bg-white border-b border-slate-200 shadow-sm">
        <div className="max-w-4xl mx-auto px-6 py-8">
          <h1 className="text-4xl font-bold text-slate-900 mb-2">mylar</h1>
          <p className="text-lg text-slate-600">
            Interactive code visualization for Git repositories{" "}
          </p>
          <p>
            <a
              href="https://github.com/chromy/mylar"
              target="_blank"
              rel="noopener noreferrer"
              className="text-blue-600 hover:text-blue-800 underline font-xxs"
            >
              https://github.com/chromy/mylar
            </a>
          </p>
        </div>
      </header>

      <main className="max-w-4xl mx-auto px-6 py-12">
        <div className="bg-white rounded-xs shadow-sm border border-slate-200 p-8">
          <div className="mb-8">
            <h2 className="text-xl font-semibold text-slate-900 mb-4">
              Explore a Repository
            </h2>
            <form onSubmit={handleGithubRepoSubmit}>
              <div className="flex gap-3">
                <input
                  type="text"
                  value={githubRepo}
                  onChange={e => setGithubRepo(e.target.value)}
                  placeholder="Enter GitHub repo (e.g., getsentry/sentry)"
                  className="flex-1 px-4 py-3 border border-slate-300 rounded-xs focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent text-slate-900 placeholder-slate-500"
                />
                <button
                  type="submit"
                  className="px-6 py-3 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-xs transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
                >
                  Visualize
                </button>
              </div>
            </form>
          </div>

          {isLoading && <FullScreenDecryptLoader />}

          {data?.repos && data.repos.length > 0 && (
            <div>
              <h2 className="text-xl font-semibold text-slate-900 mb-4">
                Recent Repositories
              </h2>
              <div className="grid gap-3">
                {data.repos.map(r => (
                  <MylarLink
                    key={r.id}
                    href={`/app/repo/${r.id}/HEAD`}
                    className="flex items-center justify-between p-4 border border-slate-200 rounded-xs hover:border-slate-300 hover:shadow-sm transition-all group"
                  >
                    <div className="flex items-center gap-3">
                      <div className="w-8 h-8 bg-slate-100 rounded-xs flex items-center justify-center">
                        <svg
                          className="w-4 h-4 text-slate-600"
                          fill="currentColor"
                          viewBox="0 0 20 20"
                        >
                          <path
                            fillRule="evenodd"
                            d="M10 0C4.477 0 0 4.484 0 10.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0110 4.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.203 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.942.359.31.678.921.678 1.856 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0020 10.017C20 4.484 15.522 0 10 0z"
                            clipRule="evenodd"
                          />
                        </svg>
                      </div>
                      <div>
                        <div className="font-medium text-slate-900 group-hover:text-blue-600 transition-colors">
                          {r.owner ?? "?"}/{r.name ?? "?"}
                        </div>
                        <div className="text-sm text-slate-500">{r.id}</div>
                      </div>
                    </div>
                    <svg
                      className="w-5 h-5 text-slate-400 group-hover:text-slate-600 transition-colors"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M9 5l7 7-7 7"
                      />
                    </svg>
                  </MylarLink>
                ))}
              </div>
            </div>
          )}
        </div>
      </main>
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
  if (!dsn || dsn === "") {
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

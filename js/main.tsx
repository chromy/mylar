import { createRoot } from 'react-dom/client';
import { Link, Route, Switch } from "wouter";

import { useState, useEffect, useCallback, useRef } from 'react';
import { z } from 'zod';

interface UseQueryResult<TData, TError> {
  data: TData | null;
  error: TError | null;
  isLoading: boolean;
  isError: boolean;
  refetch: () => void;
}

type QueryFunction<T> = (signal: AbortSignal) => Promise<T>;

interface UseQueryOptions<TData = unknown> {
  queryFn: QueryFunction<TData>,
  enabled?: boolean,
}

export function useQuery<TData = unknown, TError = Error>(
  options: UseQueryOptions<TData>,
  key: string | any[],
): UseQueryResult<TData, TError> {

  const {queryFn} = options;
  const enabled = options.enabled ?? true;

  const [data, setData] = useState<TData | null>(null);
  const [error, setError] = useState<TError | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(enabled);

  const queryFnRef = useRef(queryFn);

  useEffect(() => {
    queryFnRef.current = queryFn;
  }, [queryFn]);

  const fetchData = useCallback(async (signal?: AbortSignal) => {
    if (!enabled) {
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      const activeSignal = signal || new AbortController().signal;

      const result = await queryFnRef.current(activeSignal);

      if (!activeSignal.aborted) {
        setData(result);
        setError(null);
        setIsLoading(false);
      }
    } catch (err: any) {
      console.log("err!", err);
      if (err.name === 'AbortError') {
        return;
      }

      if (!signal?.aborted) {
        setError(err as TError);
        setData(null);
        setIsLoading(false);
      }
    }
  }, [enabled]);

  useEffect(() => {
    const controller = new AbortController();

    fetchData(controller.signal);

    return () => {
      controller.abort();
    };
  }, [fetchData, JSON.stringify(key)]);

  return {
    data,
    error,
    isLoading,
    isError: !!error,
    refetch: () => fetchData(),
  };
}

interface UseJsonQueryOptions<T> {
  path: string;
  params?: Record<string, any>;
  schema: z.ZodSchema<T>;
  enabled?: boolean;
}

interface UseJsonQueryResult<T, TError = Error> extends UseQueryResult<T, TError> {}

export function useJsonQuery<T>(
  options: UseJsonQueryOptions<T>,
  key?: string | any[]
): UseJsonQueryResult<T> {
  const { path, params, schema, enabled = true } = options;

  const queryFn = useCallback(async (signal: AbortSignal): Promise<T> => {
    const url = new URL(path, window.location.origin);

    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          url.searchParams.set(key, String(value));
        }
      });
    }

    const response = await fetch(url.toString(), { signal });

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }

    const rawJson = await response.json();

    try {
      return schema.parse(rawJson);
    } catch (error) {
      if (error instanceof z.ZodError) {
        throw new Error(`Schema validation failed: ${error.message}`);
      }
      throw error;
    }
  }, [path, params, schema]);

  const queryKey = key || [path, params];

  return useQuery<T>({
    queryFn,
    enabled
  }, queryKey);
}

async function fetchJsonQueryFn(signal: AbortSignal): Promise<unknown> {
  const response = await fetch(
    "/api/fs/get",
    { signal },
  );
  if (!response.ok) {
    throw new Error('Failed to fetch');
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
        {isLoading && (<DecryptLoader/>)}

        { data && data.repos.map(r => (<Link href="/repo/{r.name}">{r.name}</Link>) )}
      </div>
    </div>
  );
};

const Repo = () => {
  return (<div>Hello, world!</div>);
};



const App = () => (
  <>
    <Switch>
      <Route path="/" component={Home} />
      <Route path="/repo/:repo" component={Repo} />
      <Route>404: No such page!</Route>
    </Switch>
  </>
);

const DecryptLoader = () => {
  const [text, setText] = useState("LOADING");
  const originalText = "LOADING";
  const chars = "{}?0XYZ#@![]";

  useEffect(() => {
    let iteration = 0;
    const interval = setInterval(() => {
      setText(prev =>
        prev
          .split("")
          .map((letter, index) => {
            if (index < Math.floor(iteration)) {
              return originalText[index];
            }
            return chars[Math.floor(Math.random() * chars.length)];
          })
          .join("")
      );

      if (iteration >= originalText.length + 3) {
        iteration = 0; // Loop the effect
      }
      iteration += 1/3;
    }, 50);

    return () => clearInterval(interval);
  }, []);

  return (
    <div className="text-center m-2 font-mono text-green-500 text-xl font-bold">{text}</div>
  );
};


//function App() {
//
//  const { data, isLoading, isError, error } = useJsonQuery({
//    path: "/api/index/status",
//    schema: IndexStatusResponseSchema,
//  });
//
//  if (isLoading) {
//    return <div>Loading...</div>;
//  }
//  if (isError) {
//    return <div>Error {error?.toString()}</div>;
//  }
//
//  return <h1>{JSON.stringify(data)}</h1>;
//}

export function main() {
  const dom = document.querySelector("main")!;
  const root = createRoot(dom);
  root.render(<App/>);
}


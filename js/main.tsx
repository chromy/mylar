import { createRoot } from 'react-dom/client';

import { useState, useEffect, useCallback, useRef } from 'react';

interface UseQueryResult<TData, TError> {
  data: TData | null;
  error: TError | null;
  isLoading: boolean;
  isError: boolean;
  refetch: () => void;
}

type QueryFunction<T> = (signal: AbortSignal) => Promise<T>;

interface UseQueryOptions {
  queryFn: QueryFunction<TData>,
  enabled?: boolean,
}

export function useQuery<TData = unknown, TError = Error>(
  options: UseQueryOptions,
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

async function fetchJsonQueryFn(signal: AbortSignal): unknown {
  const response = await fetch(
    "/api/fs/get",
    { signal },
  );
  if (!response.ok) {
    throw new Error('Failed to fetch');
  }
  return await response.json();
}

function App() {

  const { data, isLoading, isError, error } = useQuery<User>(
    {
      queryFn: fetchQueryFn,
    },
    "metadata"
  );

  if (isLoading) {
    return <div>Loading...</div>;
  }
  if (isError) {
    return <div>Error {error.toString()}</div>;
  }

  return <h1>{JSON.stringify(data)}</h1>;
}

export function main() {
  const dom = document.querySelector("main");
  const root = createRoot(dom);
  root.render(<App/>);
}


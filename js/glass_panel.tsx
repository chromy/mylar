import { type ReactNode } from "react";

interface GlassPanelProps {
  area?: string;
  children: ReactNode;
}

export const GlassPanel = ({
  area = "mylar-content-info",
  children,
}: GlassPanelProps) => {
  return (
    <div
      className={`${area} backdrop-blur-sm bg-white/70 z-1 border border-solid rounded-xs border-black/5 m-1 p-2 text-zinc-950/80 text-xs`}
    >
      {children}
    </div>
  );
};

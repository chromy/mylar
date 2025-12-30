import { type ReactNode } from "react";
import { GlassPanel } from "./glass_panel.js";

interface ModalPanelProps {
  isOpen: boolean;
  onClose: () => void;
  title?: string;
  children: ReactNode;
}

export const ModalPanel = ({
  isOpen,
  onClose,
  title,
  children,
}: ModalPanelProps) => {
  if (!isOpen) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0" onClick={onClose} />
      <GlassPanel area="modal-content relative max-w-md w-full max-h-96 overflow-y-auto">
        <div className="flex items-center justify-between mb-4">
          {title && <h2 className="font-semibold">{title}</h2>}
          <button
            onClick={onClose}
            className="ml-auto px-2 py-1 rounded hover:bg-white/10 transition-colors"
          >
            ×
          </button>
        </div>
        {children}
      </GlassPanel>
    </div>
  );
};

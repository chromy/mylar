import { type MylarAction, type MylarState, settings } from "./state.js";
import { type ActionDispatch, useState, useEffect } from "react";
import { Command } from "cmdk";

export interface CommandMenuProps {
  dispatch: ActionDispatch<[action: MylarAction]>;
  state: MylarState;
}

export const CommandMenu = ({ dispatch, state }: CommandMenuProps) => {
  const [open, setOpen] = useState(false);

  useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setOpen(open => !open);
      }
    };

    document.addEventListener("keydown", down);
    return () => document.removeEventListener("keydown", down);
  }, []);

  return (
    <Command.Dialog
      open={open}
      onOpenChange={setOpen}
      label="Global Command Menu"
    >
      <Command.Input />
      <Command.List>
        <Command.Empty>No results found.</Command.Empty>

        <Command.Group heading="Toggle Settings">
          {settings.items.map(setting => {
            const isEnabled = setting.get(state);
            return (
              <Command.Item
                key={setting.id}
                onSelect={() => {
                  dispatch(isEnabled ? setting.disable : setting.enable);
                  setOpen(false);
                }}
              >
                {isEnabled ? "Disable" : "Enable"} {setting.name}
              </Command.Item>
            );
          })}
        </Command.Group>
      </Command.List>
    </Command.Dialog>
  );
};

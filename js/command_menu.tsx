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
          {settings.items.map(setting => (
            <Command.Item
              key={`enable-${setting.id}`}
              onSelect={() => {
                dispatch(setting.enable);
                setOpen(false);
              }}
            >
              Enable {setting.name}
            </Command.Item>
          ))}
          {settings.items.map(setting => (
            <Command.Item
              key={`disable-${setting.id}`}
              onSelect={() => {
                dispatch(setting.disable);
                setOpen(false);
              }}
            >
              Disable {setting.name}
            </Command.Item>
          ))}
        </Command.Group>
      </Command.List>
    </Command.Dialog>
  );
};

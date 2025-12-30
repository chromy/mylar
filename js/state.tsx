import { z } from "zod";
import { Registry } from "./registry.js";

export type MylarAction = { type: string };

export interface MylarState {
  [key: string]: unknown;
}

export interface SettingArgs {
  id: string;
}

export interface Setting {
  id: string;
  //  name: string;
  //  description: string;
  //  actions: MylarAction[];
  //  get(state: MylarState): T;
  //  set(state: MylarState, value: T): MylarState;
}

export class SettingsStore {
  registry: Registry<Setting>;

  constructor() {
    this.registry = new Registry();
  }

  create(args: SettingArgs): Setting {
    this.registry.register(args);
    return args;
  }

  get items(): Setting[] {
    return this.registry.items;
  }
}

export const settings = new SettingsStore();

const fpsSetting = settings.create({
  id: "fps",
});

export const mylarReducer = (
  state: MylarState,
  action: MylarAction,
): MylarState => {
  switch (action.type) {
    case "SHOW_FPS_COUNTER":
      return { ...state, showFpsCounter: true };
    case "HIDE_FPS_COUNTER":
      return { ...state, showFpsCounter: false };
    case "TOGGLE_SETTINGS":
      return {
        ...state,
        showSettings: !(state as any).showSettings,
        showHelp: false,
      };
    case "TOGGLE_HELP":
      return {
        ...state,
        showHelp: !(state as any).showHelp,
        showSettings: false,
      };
    case "CLOSE_ALL_PANELS":
      return { ...state, showSettings: false, showHelp: false };
    default:
      return state;
  }
};

export const initialMylarState: MylarState = {
  showSettings: false,
  showHelp: false,
  showFpsCounter: false,
};

import { z } from "zod";
import { Registry } from "./registry.js";

export interface ChangeSettingMylarAction {
  type: "CHANGE_SETTING";
  id: string;
  value: unknown;
}

export interface ChangeLayerMylarAction {
  type: "CHANGE_LAYER";
  layer: LayerType;
}

export type MylarAction = ChangeSettingMylarAction | ChangeLayerMylarAction;

export interface MylarState {
  [key: string]: unknown;
}

export interface SettingArgs {
  id: string;
  name?: string;
  defaultValue?: boolean;
}

export interface SettingAction {
  id: string;
  name: string;
}

export interface Setting {
  id: string;
  name: string;
  //  description: string;
  get(state: MylarState): boolean;
  set(state: MylarState, rawValue: unknown): MylarState;
  enable: MylarAction;
  disable: MylarAction;
}

export class SettingsStore {
  registry: Registry<Setting>;

  constructor() {
    this.registry = new Registry();
    //this.actions = new Registry();
  }

  addBoolean(args: SettingArgs): Setting {
    const schema = z.boolean().default(args.defaultValue ?? false);
    const s: Setting = {
      id: args.id,
      name: args.name ?? args.id,
      get(state: MylarState): boolean {
        return schema.parse(state[this.id]);
      },
      set(state: MylarState, rawValue: unknown): MylarState {
        const next = { ...state };
        next[args.id] = schema.parse(rawValue);
        return next;
      },
      enable: {
        type: "CHANGE_SETTING",
        id: args.id,
        value: true,
      },
      disable: {
        type: "CHANGE_SETTING",
        id: args.id,
        value: false,
      },
    };
    this.registry.register(s);
    return s;
  }

  get items(): Setting[] {
    return this.registry.items;
  }

  get(id: string): Setting {
    const maybeSetting = this.registry.items.filter(item => item.id === id)[0];
    if (maybeSetting === undefined) {
      throw new Error(`No such setting ${maybeSetting}`);
    }
    return maybeSetting;
  }
}

export const settings = new SettingsStore();

export const settingsPanelSetting = settings.addBoolean({
  id: "setting.settingsPanel",
  name: "settings panel",
});

export interface LayerType {
  kind: string;
  composite: string;
}

export const mylarReducer = (
  state: MylarState,
  action: MylarAction,
): MylarState => {
  switch (action.type) {
    case "CHANGE_SETTING":
      return settings.get(action.id).set(state, action.value);
    case "CHANGE_LAYER":
      return { ...state, layer: action.layer };
    default:
      return state;
  }
};

export const initialMylarState: MylarState = {
  layer: { kind: "fileExtension", composite: "direct" } as LayerType,
};

export const getCurrentLayer = (state: MylarState): LayerType => {
  return (state.layer as LayerType) ?? { kind: "fileExtension", composite: "direct" };
};

export const createChangeLayerAction = (
  layer: LayerType,
): ChangeLayerMylarAction => ({
  type: "CHANGE_LAYER",
  layer,
});

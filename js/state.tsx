export interface MylarState {
  showSettings: boolean;
  showHelp: boolean;
  showFpsCounter: boolean;
}

export type MylarAction =
  | { type: "TOGGLE_SETTINGS" }
  | { type: "SHOW_FPS_COUNTER" }
  | { type: "HIDE_FPS_COUNTER" }
  | { type: "TOGGLE_HELP" }
  | { type: "CLOSE_ALL_PANELS" };

export const mylarReducer = (state: MylarState, action: MylarAction): MylarState => {
  switch (action.type) {
    case "SHOW_FPS_COUNTER":
      return { ...state, showFpsCounter: true };
    case "HIDE_FPS_COUNTER":
      return { ...state, showFpsCounter: false };
    case "TOGGLE_SETTINGS":
      return { ...state, showSettings: !state.showSettings, showHelp: false };
    case "TOGGLE_HELP":
      return { ...state, showHelp: !state.showHelp, showSettings: false };
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


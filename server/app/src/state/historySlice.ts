import { createAsyncThunk, createSlice, PayloadAction } from "@reduxjs/toolkit";
import { BookDetail, ParseError } from "./messages";
import { setActiveItem } from "./stateSlice";
import { AppDispatch, RootState } from "./store";

// HistoryItem represents a single search history item
type HistoryItem = {
  query: string;
  timestamp: number;
  results?: BookDetail[];
  errors?: ParseError[];
};

interface HistoryState {
  items: HistoryItem[];
}

// A search is pending until its results (or failure) arrive.
const isPending = (item: HistoryItem): boolean => item.results === undefined;

const loadState = (): HistoryItem[] => {
  try {
    const items: HistoryItem[] =
      JSON.parse(localStorage.getItem("history")!) ?? [];
    // Searches from a previous session can never receive their results.
    return items.filter((item) => !isPending(item));
  } catch (err) {
    return [];
  }
};

const initialState: HistoryState = {
  items: loadState()
};

export const historySlice = createSlice({
  name: "history",
  initialState,
  reducers: {
    addHistoryItem: (state, action: PayloadAction<HistoryItem>) => {
      state.items = [action.payload, ...state.items].slice(0, 16);
    },
    deleteByTimetamp: (state, action: PayloadAction<number>) => {
      state.items = state.items.filter((x) => x.timestamp !== action.payload);
    },
    dropPendingHistoryItems: (state) => {
      state.items = state.items.filter((item) => !isPending(item));
    },
    updateHistoryItem: (state, action: PayloadAction<HistoryItem>) => {
      var pendingItemIndex = state.items.findIndex(
        (x) => x.timestamp === action.payload.timestamp
      );
      if (pendingItemIndex === -1) {
        return;
      }
      state.items = [
        ...state.items.slice(0, pendingItemIndex),
        action.payload,
        ...state.items.slice(pendingItemIndex + 1)
      ];
    }
  }
});

// Delete an item from history. Clear current item and loading state if deleting active search
const deleteHistoryItem = createAsyncThunk<
  Promise<void>,
  number | undefined,
  { dispatch: AppDispatch; state: RootState }
>("history/delete_item", async (timeStamp, { dispatch, getState }) => {
  if (timeStamp === undefined) {
    dispatch(setActiveItem(null));
    const toRemove = getState().history.items.at(0)?.timestamp;
    if (toRemove) {
      dispatch(historySlice.actions.deleteByTimetamp(toRemove));
    }
    return;
  }

  const activeItem = getState().state.activeItem;
  if (activeItem?.timestamp === timeStamp) {
    dispatch(setActiveItem(null));
  }

  dispatch(historySlice.actions.deleteByTimetamp(timeStamp));
});

const { addHistoryItem, updateHistoryItem, dropPendingHistoryItems } =
  historySlice.actions;

const selectHistory = (state: RootState) => state.history.items;

export type { HistoryItem };
export {
  deleteHistoryItem,
  addHistoryItem,
  updateHistoryItem,
  dropPendingHistoryItems,
  selectHistory,
  isPending
};

export default historySlice.reducer;

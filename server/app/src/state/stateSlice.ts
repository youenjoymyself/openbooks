import {
  createAction,
  createAsyncThunk,
  createSlice,
  PayloadAction
} from "@reduxjs/toolkit";
import {
  addHistoryItem,
  dropPendingHistoryItems,
  HistoryItem,
  isPending,
  updateHistoryItem
} from "./historySlice";
import { MessageType, SearchResponse } from "./messages";
import { AppDispatch, RootState } from "./store";

interface AppState {
  isConnected: boolean;
  isSidebarOpen: boolean;
  activeItem: HistoryItem | null;
  username?: string;
  inFlightDownloads: string[];
  // Minimum seconds between searches, sent by the server on connect.
  searchTimeout: number;
  // Epoch milliseconds when the server will accept the next search.
  nextSearchAt: number;
}

const loadActive = (): HistoryItem | null => {
  try {
    const active: HistoryItem | null = JSON.parse(
      localStorage.getItem("active")!
    );
    // A search from a previous session can never receive its results.
    // (isPending isn't usable here: this runs during the circular import.)
    return active?.results !== undefined ? active : null;
  } catch (err) {
    return null;
  }
};

const initialState: AppState = {
  isConnected: false,
  isSidebarOpen: true,
  activeItem: loadActive(),
  username: undefined,
  inFlightDownloads: [],
  searchTimeout: 0,
  nextSearchAt: 0
};

const stateSlice = createSlice({
  name: "state",
  initialState,
  reducers: {
    setActiveItem(state, action: PayloadAction<HistoryItem | null>) {
      state.activeItem = action.payload;
    },
    setConnectionState(state, action: PayloadAction<boolean>) {
      state.isConnected = action.payload;
    },
    setUsername(state, action: PayloadAction<string>) {
      state.username = action.payload;
    },
    addInFlightDownload(state, action: PayloadAction<string>) {
      state.inFlightDownloads.push(action.payload);
    },
    // Remove the in-flight download that produced fileName. Falls back to the
    // oldest download when fileName is missing (failures don't name the book)
    // or doesn't appear in any requested book string.
    removeInFlightDownload(state, action: PayloadAction<string | undefined>) {
      const index = state.inFlightDownloads.findIndex((book) =>
        matchesDownload(book, action.payload)
      );
      state.inFlightDownloads.splice(Math.max(index, 0), 1);
    },
    clearInFlightDownloads(state) {
      state.inFlightDownloads = [];
    },
    setSearchTimeout(state, action: PayloadAction<number>) {
      state.searchTimeout = action.payload;
    },
    // Block searches for the given number of seconds.
    delayNextSearch(state, action: PayloadAction<number>) {
      state.nextSearchAt = Date.now() + action.payload * 1000;
    },
    toggleSidebar(state) {
      state.isSidebarOpen = !state.isSidebarOpen;
    }
  }
});

// matchesDownload reports whether the requested book string produced the
// received file. The extension is ignored because archives are extracted.
const matchesDownload = (book: string, fileName?: string): boolean => {
  const base = fileName
    ?.split(/[\\/]/)
    .pop()
    ?.replace(/\.[^.]+$/, "");
  return !!base && book.includes(base);
};

// Action that sends a websocket message to the server
const sendMessage = createAction("socket/send_message", (message: any) => ({
  payload: { message: JSON.stringify(message) }
}));

const sendDownload = createAsyncThunk(
  "state/send_download",
  (book: string, { dispatch }) => {
    dispatch(addInFlightDownload(book));
    dispatch(
      sendMessage({
        type: MessageType.DOWNLOAD,
        payload: { book }
      })
    );
  }
);

// Send a search to the server. Add to query history and set loading.
const sendSearch = createAsyncThunk<
  void,
  string,
  { dispatch: AppDispatch; state: RootState }
>("state/send_sendSearch", (queryString: string, { dispatch, getState }) => {
  // Send the books search query to the server
  dispatch(
    sendMessage({
      type: MessageType.SEARCH,
      payload: {
        query: queryString
      }
    })
  );

  dispatch(delayNextSearch(getState().state.searchTimeout));

  const timestamp = new Date().getTime();

  // Add query to item history.
  dispatch(addHistoryItem({ query: queryString, timestamp }));
  dispatch(setActiveItem({ query: queryString, timestamp: timestamp }));
});

const setSearchResults = createAsyncThunk<
  Promise<void>,
  SearchResponse,
  { dispatch: AppDispatch; state: RootState }
>(
  "state/set_search_results",
  async ({ books, errors }: SearchResponse, { dispatch, getState }) => {
    // IRC answers searches in order, so results belong to the oldest pending
    // search. History is stored newest first.
    const pending = getState().history.items.filter(isPending).at(-1);
    if (pending === undefined) {
      return;
    }
    const updatedItem: HistoryItem = {
      query: pending.query,
      timestamp: pending.timestamp,
      results: books ?? [],
      errors: errors ?? []
    };

    if (getState().state.activeItem?.timestamp === pending.timestamp) {
      dispatch(setActiveItem(updatedItem));
    }
    dispatch(updateHistoryItem(updatedItem));
  }
);

// Drop everything waiting on the IRC session after the websocket closes. The
// next connection gets a new IRC session that can't answer them.
const resetPendingRequests = createAsyncThunk<
  void,
  void,
  { dispatch: AppDispatch; state: RootState }
>("state/reset_pending", (_, { dispatch, getState }) => {
  const activeItem = getState().state.activeItem;
  if (activeItem && isPending(activeItem)) {
    dispatch(setActiveItem(null));
  }
  dispatch(dropPendingHistoryItems());
  dispatch(clearInFlightDownloads());
});

export const {
  setActiveItem,
  setConnectionState,
  setUsername,
  addInFlightDownload,
  removeInFlightDownload,
  clearInFlightDownloads,
  setSearchTimeout,
  delayNextSearch,
  toggleSidebar
} = stateSlice.actions;

export {
  stateSlice,
  sendMessage,
  sendDownload,
  sendSearch,
  setSearchResults,
  resetPendingRequests
};

export default stateSlice.reducer;

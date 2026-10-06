import {
  AnyAction,
  Dispatch,
  Middleware,
  MiddlewareAPI,
  PayloadAction
} from "@reduxjs/toolkit";
import { openbooksApi } from "./api";
import { deleteHistoryItem } from "./historySlice";
import {
  ConnectionResponse,
  DownloadResponse,
  MessageType,
  Notification,
  NotificationType,
  RateLimitResponse,
  Response,
  SearchResponse
} from "./messages";
import { addNotification } from "./notificationSlice";
import {
  delayNextSearch,
  removeInFlightDownload,
  resetPendingRequests,
  sendMessage,
  setConnectionState,
  setSearchResults,
  setSearchTimeout,
  setUsername
} from "./stateSlice";
import { AppDispatch, RootState } from "./store";
import { displayNotification, downloadFile } from "./util";

const minRetryDelay = 1_000;
const maxRetryDelay = 30_000;

// Web socket redux middleware.
// Listens to socket and dispatches handlers.
// Handles send_message actions by sending to socket.
// Reconnects with exponential backoff when the socket closes.
export const websocketConn =
  (wsUrl: string): Middleware =>
  ({ dispatch, getState }: MiddlewareAPI<AppDispatch, RootState>) => {
    let socket: WebSocket;
    let retryDelay = minRetryDelay;
    let outageNotified = false;

    const connect = () => {
      socket = new WebSocket(wsUrl);

      socket.onopen = () => {
        retryDelay = minRetryDelay;
        outageNotified = false;
        onOpen(dispatch);
      };
      socket.onclose = () => {
        onClose(dispatch);
        // Only notify once per outage, not on every retry.
        if (!outageNotified) {
          outageNotified = true;
          displayNotification({
            appearance: NotificationType.DANGER,
            title: "Unable to connect to server. Reconnecting…",
            timestamp: new Date().getTime()
          });
        }
        setTimeout(connect, retryDelay);
        retryDelay = Math.min(retryDelay * 2, maxRetryDelay);
      };
      socket.onmessage = (message) => route(dispatch, message);
    };

    connect();

    return (next: Dispatch<AnyAction>) => (action: PayloadAction<any>) => {
      // Send Message action? Send data to the socket.
      if (sendMessage.match(action)) {
        if (socket.readyState === socket.OPEN) {
          socket.send(action.payload.message);
        } else {
          displayNotification({
            appearance: NotificationType.WARNING,
            title: "Not connected to the server. Reconnecting…",
            timestamp: new Date().getTime()
          });
        }
      }

      return next(action);
    };
  };

const onOpen = (dispatch: AppDispatch): void => {
  console.log("WebSocket connected.");
  dispatch(setConnectionState(true));
  dispatch(sendMessage({ type: MessageType.CONNECT, payload: {} }));
};

const onClose = (dispatch: AppDispatch): void => {
  console.log("WebSocket closed.");
  dispatch(setConnectionState(false));
  dispatch(resetPendingRequests());
};

const route = (dispatch: AppDispatch, msg: MessageEvent<any>): void => {
  const getNotif = (): Notification => {
    let response = JSON.parse(msg.data) as Response;
    const timestamp = new Date().getTime();
    const notification: Notification = {
      ...response,
      timestamp
    };

    switch (response.type) {
      case MessageType.STATUS:
        return notification;
      case MessageType.CONNECT:
        dispatch(setUsername((response as ConnectionResponse).name));
        dispatch(
          setSearchTimeout((response as ConnectionResponse).searchTimeout ?? 0)
        );
        return notification;
      case MessageType.SEARCH:
        dispatch(setSearchResults(response as SearchResponse));
        return notification;
      case MessageType.DOWNLOAD: {
        // Failures arrive as DOWNLOAD messages too, without a file.
        const succeeded = response.appearance === NotificationType.SUCCESS;
        if (succeeded) {
          downloadFile((response as DownloadResponse).downloadPath);
          dispatch(openbooksApi.util.invalidateTags(["books"]));
        }
        dispatch(
          removeInFlightDownload(succeeded ? response.detail : undefined)
        );
        return notification;
      }
      case MessageType.RATELIMIT:
        dispatch(deleteHistoryItem());
        dispatch(
          delayNextSearch((response as RateLimitResponse).retryAfter ?? 0)
        );
        return notification;
      default:
        console.error(response);
        return {
          appearance: NotificationType.DANGER,
          title: "Unknown message type. See console.",
          timestamp
        };
    }
  };

  const notif = getNotif();
  dispatch(addNotification(notif));
  displayNotification(notif);
};

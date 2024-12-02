import { roomState, connected, errorMessage } from '../stores/room';

export function createWebSocket(roomId, name, isGameMaster) {
  const ws = new WebSocket(
    `wss://${window.location.host}/api/ws?roomId=${roomId}&name=${name}&gamemaster=${isGameMaster}`
  );

  ws.onmessage = (event) => {
    try {
      const message = JSON.parse(event.data);
      if (message.type === "roomState") {
        roomState.set(message.payload);
        connected.set(true);
        errorMessage.set("");
      } else if (message.error) {
        errorMessage.set(message.error);
      }
    } catch (error) {
      errorMessage.set("Error processing message from server");
    }
  };

  ws.onerror = () => {
    errorMessage.set("Connection error occurred");
    connected.set(false);
  };

  return ws;
}

export async function createRoom(tickets) {
    const response = await fetch(`https://${window.location.host}/api/create-room`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        ticketIds: tickets.split("\n").filter((t) => t.trim()),
      }),
    });
  
    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(errorText || "Failed to create room");
    }
  
    return response.json();
  }
  
  export async function destroyRoom(roomId, name) {
    const response = await fetch(
      `https://${window.location.host}/api/destroy-room?roomId=${roomId}&name=${name}`,
      { method: "DELETE" }
    );
  
    if (!response.ok) {
      throw new Error("Failed to destroy room");
    }
  }
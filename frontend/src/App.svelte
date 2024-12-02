<script>
  import { onMount } from "svelte";
  import { version } from "../package.json";

  let name = "";
  let tickets = "";
  let roomId = "";
  let ws;
  let roomState = null;
  let creating = true;
  let connected = false;
  let isGameMaster = false;
  let errorMessage = "";
  let connecting = false;

  onMount(() => {
    const params = new URLSearchParams(window.location.search);
    const roomIdParam = params.get("roomId");
    if (roomIdParam) {
      roomId = roomIdParam;
      creating = false;
    }
  });

  async function createRoom() {
    try {
      connecting = true;
      errorMessage = "";

      if (!tickets.trim()) {
        throw new Error("Please enter at least one ticket ID");
      }

      const response = await fetch(
        `https://${window.location.host}/api/create-room`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            ticketIds: tickets.split("\n").filter((t) => t.trim()),
          }),
        },
      );

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(errorText || "Failed to create room");
      }

      const data = await response.json();
      roomId = data.roomId;
      isGameMaster = true;
      connectWebSocket();
      window.history.pushState({}, "", `?roomId=${roomId}`);
    } catch (error) {
      errorMessage = `Error: ${error.message}`;
      connected = false;
    } finally {
      connecting = false;
    }
  }

  function connectWebSocket() {
    try {
      connecting = true;
      errorMessage = "";

      if (!name.trim()) {
        throw new Error("Please enter your name");
      }

      ws = new WebSocket(
        `wss://${window.location.host}/api/ws?roomId=${roomId}&name=${name}&gamemaster=${isGameMaster}`,
      );

      ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data);
          if (message.type === "roomState") {
            roomState = message.payload;
            connected = true;
            errorMessage = "";
          } else if (message.error) {
            errorMessage = message.error;
          }
        } catch (error) {
          errorMessage = "Error processing message from server";
        }
      };

      ws.onerror = () => {
        errorMessage = "Connection error occurred";
        connected = false;
      };

      ws.onclose = () => {
        connected = false;
        if (!errorMessage) {
          errorMessage = "Connection lost. Reconnecting...";
          setTimeout(connectWebSocket, 1000);
        }
      };
    } catch (error) {
      errorMessage = `Error: ${error.message}`;
      connected = false;
    } finally {
      connecting = false;
    }
  }

  function vote(ticketId, score) {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      errorMessage = "Connection lost. Please refresh the page.";
      return;
    }

    ws.send(
      JSON.stringify({
        type: "vote",
        payload: { ticketId, vote: score },
      }),
    );
  }

  function revealVotes() {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      errorMessage = "Connection lost. Please refresh the page.";
      return;
    }

    ws.send(JSON.stringify({ type: "reveal" }));
  }

  function nextTicket() {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      errorMessage = "Connection lost. Please refresh the page.";
      return;
    }

    ws.send(JSON.stringify({ type: "next" }));
  }

  function joinRoom() {
    connectWebSocket();
    creating = false;
    window.history.pushState({}, "", `?roomId=${roomId}`);
  }

  function previousTicket() {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      errorMessage = "Connection lost. Please refresh the page.";
      return;
    }
    ws.send(JSON.stringify({ type: "previous" }));
  }

  async function destroyRoom() {
    try {
      const response = await fetch(
        `https://${window.location.host}/api/destroy-room?roomId=${roomId}&name=${name}`,
        { method: "DELETE" },
      );

      if (!response.ok) {
        throw new Error("Failed to destroy room");
      }

      window.location.href = window.location.pathname;
    } catch (error) {
      errorMessage = `Error: ${error.message}`;
    }
  }
</script>

<div class="version">v{version}</div>

{#if !connected}
  <div class="app">
    <div class="login-container">
      {#if errorMessage}
        <div class="error-message">
          {errorMessage}
        </div>
      {/if}

      {#if connecting}
        <div class="connecting-message">Connecting...</div>
      {/if}

      <input
        type="text"
        bind:value={name}
        placeholder="Enter your name"
        class="login-input"
      />

      {#if creating}
        <textarea
          bind:value={tickets}
          placeholder="Enter ticket IDs (one per line)"
          class="login-input"
          rows="5"
        ></textarea>
        <div class="button-group">
          <button
            on:click={createRoom}
            disabled={!name || !tickets || connecting}
            class="primary-button"
          >
            Create Room
          </button>
          <button
            on:click={() => (creating = false)}
            class="secondary-button"
            disabled={connecting}
          >
            Join Existing Room
          </button>
        </div>
      {:else}
        <input
          type="text"
          bind:value={roomId}
          placeholder="Enter room ID"
          class="login-input"
        />
        <div class="button-group">
          <button
            on:click={joinRoom}
            disabled={!name || !roomId || connecting}
            class="primary-button"
          >
            Join Room
          </button>
          <button
            on:click={() => (creating = true)}
            class="secondary-button"
            disabled={connecting}
          >
            Create New Room
          </button>
        </div>
      {/if}
    </div>
  </div>
{:else}
  <div class="app">
    {#if errorMessage}
      <div class="error-message">
        {errorMessage}
      </div>
    {/if}

    <div class="room-header">
      <div class="room-info">
        Room ID: <span class="room-id">{roomId}</span>
        <button
          class="copy-button"
          on:click={() => {
            navigator.clipboard.writeText(window.location.href);
          }}
        >
          Copy Link
        </button>
      </div>
      <div class="participants">
        <div class="game-master">
          Game Master: {roomState.GameMaster}
          {#if roomState.GameMaster === name}(You){/if}
        </div>
        <div class="players">
          Players: {Object.keys(roomState.Users)
            .filter((u) => u !== roomState.GameMaster)
            .join(", ")}
        </div>
      </div>
    </div>

    {#if roomState.Tickets[roomState.CurrentTicket]}
      <div class="ticket-card">
        <h3>Current Ticket: {roomState.Tickets[roomState.CurrentTicket].ID}</h3>
        <div class="ticket-progress">
          Ticket {roomState.CurrentTicket + 1} of {roomState.Tickets.length}
        </div>

        {#if !isGameMaster}
          <div class="voting-panel">
            {#each [0, 1, 2, 3, 5, 8, 13, 20, 40, 100, "?"] as score}
              <button
                class="vote-button {roomState.Tickets[roomState.CurrentTicket]
                  .Votes[name] === score
                  ? 'selected'
                  : ''}"
                on:click={() =>
                  vote(roomState.Tickets[roomState.CurrentTicket].ID, score)}
              >
                {score}
              </button>
            {/each}
          </div>
        {/if}

        <div class="votes-section">
          {#if roomState.VotesRevealed}
            <h4>Votes:</h4>
            {#each Object.entries(roomState.Tickets[roomState.CurrentTicket].Votes) as [user, vote]}
              <div class="vote-entry">{user}: {vote}</div>
            {/each}
          {:else}
            <div class="hidden-votes">
              {Object.keys(roomState.Tickets[roomState.CurrentTicket].Votes)
                .length} votes cast
            </div>
          {/if}
        </div>

        {#if isGameMaster}
          <div class="control-panel">
            <button
              on:click={revealVotes}
              disabled={roomState.VotesRevealed}
              class="reveal-button"
            >
              Reveal Votes
            </button>
            <button
              on:click={previousTicket}
              disabled={roomState.CurrentTicket <= 0}
              class="previous-button"
            >
              Previous
            </button>
            <button
              on:click={nextTicket}
              disabled={roomState.CurrentTicket >= roomState.Tickets.length - 1}
              class="next-button"
            >
              Next
            </button>
            <button on:click={destroyRoom} class="destroy-button">
              Destroy Room
            </button>
          </div>
        {/if}
      </div>
    {/if}
  </div>
{/if}

<style>
  .app {
    background: #1e1e1e;
    color: #fff;
    padding: 2rem;
    /* min-height: 100vh; */
  }

  .login-container {
    max-width: 500px;
    margin: 0 auto;
    padding: 2rem;
  }

  .error-message {
    background: #ff44336e;
    color: white;
    padding: 1rem;
    border-radius: 4px;
    margin: 1rem 0;
  }

  .connecting-message {
    color: #4caf50;
    text-align: center;
    margin: 1rem 0;
  }

  .login-input {
    width: 100%;
    padding: 0.75rem;
    margin-bottom: 1rem;
    background: #2d2d2d;
    border: 1px solid #3d3d3d;
    color: #fff;
    border-radius: 4px;
  }

  .login-input:focus {
    outline: none;
    border-color: #4caf50;
  }

  .button-group {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1rem;
    margin-top: 1rem;
  }

  .primary-button,
  .secondary-button {
    padding: 0.75rem;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    font-weight: 500;
    transition: opacity 0.2s;
  }

  .primary-button {
    background: #4caf50;
    color: white;
  }

  .secondary-button {
    background: #2d2d2d;
    color: #fff;
  }

  .primary-button:disabled,
  .secondary-button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .room-header {
    background: #2d2d2d;
    padding: 1.5rem;
    border-radius: 8px;
    margin-bottom: 1.5rem;
  }

  .room-info {
    display: flex;
    align-items: center;
    gap: 1rem;
  }

  .room-id {
    color: #4caf50;
    font-family: monospace;
  }

  .copy-button {
    background: #4caf50;
    color: white;
    border: none;
    padding: 0.5rem 1rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.9rem;
  }

  .participants {
    margin-top: 1rem;
  }

  .game-master {
    color: #4caf50;
    margin-bottom: 0.5rem;
  }

  .ticket-card {
    background: #2d2d2d;
    padding: 1.5rem;
    border-radius: 8px;
  }

  .ticket-progress {
    color: #888;
    margin: 1rem 0;
  }

  .voting-panel {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(60px, 1fr));
    gap: 0.75rem;
    margin: 1.5rem 0;
  }

  .vote-button {
    aspect-ratio: 1;
    border: 2px solid #4caf50;
    background: transparent;
    color: #4caf50;
    border-radius: 50%;
    font-size: 1.1rem;
    transition: all 0.2s;
  }

  .vote-button:hover {
    background: #4caf5022;
  }

  .vote-button.selected {
    background: #4caf50;
    color: white;
  }

  .votes-section {
    margin-top: 1.5rem;
    padding-top: 1rem;
    border-top: 1px solid #3d3d3d;
  }

  .vote-entry {
    padding: 0.5rem 0;
    border-bottom: 1px solid #3d3d3d;
  }

  .hidden-votes {
    text-align: center;
    color: #888;
    font-style: italic;
  }

  .control-panel {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 1rem;
    margin-top: 1.5rem;
  }

  .previous-button {
    background: #666;
    color: white;
    padding: 0.75rem;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    font-weight: 500;
  }

  .previous-button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .reveal-button,
  .next-button {
    padding: 0.75rem;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    font-weight: 500;
  }

  .reveal-button {
    background: #666;
    color: white;
  }

  .next-button {
    background: #4caf50;
    color: white;
  }

  .reveal-button:disabled,
  .next-button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .version {
    position: fixed;
    bottom: 1rem;
    right: 1rem;
    font-size: 0.8rem;
    color: #666;
  }
  .destroy-button {
    background: #dc3545;
    color: white;
    padding: 0.75rem;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    font-weight: 500;
  }

  .destroy-button:hover {
    background: #bb2d3b;
  }
</style>

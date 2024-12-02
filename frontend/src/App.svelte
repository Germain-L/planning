<script>
  import { onMount } from "svelte";
  import { version } from "../package.json";
  import { roomState, connected, errorMessage } from "./lib/stores/room";
  import { createWebSocket } from "./lib/ws/socket";
  import { createRoom, destroyRoom } from "./lib/api/room";
  import LoginForm from "./lib/components/LoginForm.svelte";
  import RoomHeader from "./lib/components/RoomHeader.svelte";
  import VotingPanel from "./lib/components/VotingPanel.svelte";
  import ControlPanel from "./lib/components/ControlPanel.svelte";

  let name = "";
  let tickets = "";
  let roomId = "";
  let ws;
  let creating = true;
  let isGameMaster = false;
  let connecting = false;

  onMount(() => {
    const params = new URLSearchParams(window.location.search);
    const roomIdParam = params.get("roomId");
    if (roomIdParam) {
      roomId = roomIdParam;
      creating = false;
    }
  });

  async function handleCreateRoom() {
    try {
      connecting = true;
      errorMessage.set("");

      if (!tickets.trim()) {
        throw new Error("Please enter at least one ticket ID");
      }

      const data = await createRoom(tickets);
      roomId = data.roomId;
      isGameMaster = true;
      connectWebSocket();
      window.history.pushState({}, "", `?roomId=${roomId}`);
    } catch (error) {
      errorMessage.set(`Error: ${error.message}`);
      connected.set(false);
    } finally {
      connecting = false;
    }
  }

  function connectWebSocket() {
    try {
      connecting = true;
      errorMessage.set("");

      if (!name.trim()) {
        throw new Error("Please enter your name");
      }

      ws = createWebSocket(roomId, name, isGameMaster);

      ws.onclose = () => {
        connected.set(false);
        if (!$errorMessage) {
          errorMessage.set("Connection lost. Reconnecting...");
          setTimeout(connectWebSocket, 1000);
        }
      };
    } catch (error) {
      errorMessage.set(`Error: ${error.message}`);
      connected.set(false);
    } finally {
      connecting = false;
    }
  }

  function handleJoinRoom() {
    connectWebSocket();
    creating = false;
    window.history.pushState({}, "", `?roomId=${roomId}`);
  }

  function sendMessage(type, payload = {}) {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      errorMessage.set("Connection lost. Please refresh the page.");
      return;
    }
    ws.send(JSON.stringify({ type, payload }));
  }

  function vote(ticketId, score) {
    sendMessage("vote", { ticketId, vote: score });
  }

  async function handleDestroyRoom() {
    try {
      await destroyRoom(roomId, name);
      window.location.href = window.location.pathname;
    } catch (error) {
      errorMessage.set(`Error: ${error.message}`);
    }
  }
</script>

<div class="version">v{version}</div>

{#if !$connected}
  <div class="app">
    {#if $errorMessage}
      <div class="error-message">{$errorMessage}</div>
    {/if}

    {#if connecting}
      <div class="connecting-message">Connecting...</div>
    {/if}

    <LoginForm
      {name}
      {tickets}
      {roomId}
      {creating}
      {connecting}
      onCreateRoom={handleCreateRoom}
      onJoinRoom={handleJoinRoom}
    />
  </div>
{:else}
  <div class="app">
    {#if $errorMessage}
      <div class="error-message">{$errorMessage}</div>
    {/if}

    <RoomHeader {roomId} roomState={$roomState} currentUser={name} />

    {#if $roomState.Tickets[$roomState.CurrentTicket]}
      <div class="ticket-card">
        <h3>
          Current Ticket: {$roomState.Tickets[$roomState.CurrentTicket].ID}
        </h3>
        <div class="ticket-progress">
          Ticket {$roomState.CurrentTicket + 1} of {$roomState.Tickets.length}
        </div>

        {#if !isGameMaster}
          <VotingPanel
            currentTicket={$roomState.Tickets[$roomState.CurrentTicket]}
            currentUser={name}
            onVote={vote}
          />
        {/if}

        <div class="votes-section">
          {#if $roomState.VotesRevealed}
            <h4>Votes:</h4>
            {#each Object.entries($roomState.Tickets[$roomState.CurrentTicket].Votes) as [user, vote]}
              <div class="vote-entry">{user}: {vote}</div>
            {/each}
          {:else}
            <div class="hidden-votes">
              {Object.keys($roomState.Tickets[$roomState.CurrentTicket].Votes)
                .length} votes cast
            </div>
          {/if}
        </div>

        {#if isGameMaster}
          <ControlPanel
            roomState={$roomState}
            onReveal={() => sendMessage("reveal")}
            onPrevious={() => sendMessage("previous")}
            onNext={() => sendMessage("next")}
            onDestroy={handleDestroyRoom}
          />
        {/if}
      </div>
    {/if}
  </div>
{/if}

<style>
  /* Keep app-level styles only */
  .app {
    background: #1e1e1e;
    color: #fff;
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

  .version {
    position: fixed;
    bottom: 1rem;
    right: 1rem;
    font-size: 0.8rem;
    color: #666;
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
</style>

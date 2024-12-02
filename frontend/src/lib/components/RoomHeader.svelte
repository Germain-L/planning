<script>
    export let roomId;
    export let roomState;
    export let currentUser;
</script>

<div class="room-header">
    <div class="room-info">
        Room ID: <span class="room-id">{roomId}</span>
        <button
            class="copy-button"
            on:click={() => navigator.clipboard.writeText(window.location.href)}
        >
            Copy Link
        </button>
    </div>
    <div class="participants">
        <div class="game-master">
            Game Master: {roomState.GameMaster}
            {#if roomState.GameMaster === currentUser}(You){/if}
        </div>
        <div class="players">
            Players: {Object.keys(roomState.Users)
                .filter((u) => u !== roomState.GameMaster)
                .join(", ")}
        </div>
    </div>
</div>

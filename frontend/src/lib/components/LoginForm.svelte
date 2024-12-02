<script>
    export let name = "";
    export let tickets = "";
    export let roomId = "";
    export let creating = true;
    export let connecting = false;
    export let onCreateRoom;
    export let onJoinRoom;

    $: canSubmit = creating
        ? name && tickets && !connecting
        : name && roomId && !connecting;
</script>

<div class="login-container">
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
                on:click={onCreateRoom}
                disabled={!canSubmit}
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
                on:click={onJoinRoom}
                disabled={!canSubmit}
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

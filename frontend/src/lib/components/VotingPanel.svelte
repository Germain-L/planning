<script>
    export let currentTicket;
    export let currentUser;
    export let onVote;

    const scores = [0, 1, 2, 3, 5, 8, 13, 20, 40, 100];
</script>

<div class="voting-panel">
    {#each scores as score}
        <button
            class="vote-button {currentTicket.Votes[currentUser] === score
                ? 'selected'
                : ''}"
            on:click={() => onVote(currentTicket.ID, score)}
        >
            {score}
        </button>
    {/each}
</div>

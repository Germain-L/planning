<script>
    export let roomState;
    export let onReveal;
    export let onPrevious;
    export let onNext;
    export let onDestroy;
</script>

<div class="control-panel">
    <button
        on:click={onReveal}
        disabled={roomState.VotesRevealed}
        class="reveal-button"
    >
        Reveal Votes
    </button>
    <button
        on:click={onPrevious}
        disabled={roomState.CurrentTicket <= 0}
        class="previous-button"
    >
        Previous
    </button>
    <button
        on:click={onNext}
        disabled={roomState.CurrentTicket >= roomState.Tickets.length - 1}
        class="next-button"
    >
        Next
    </button>
    <button on:click={onDestroy} class="destroy-button"> Destroy Room </button>
</div>

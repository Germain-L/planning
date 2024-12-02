import { writable } from 'svelte/store';

export const roomState = writable(null);
export const connected = writable(false);
export const errorMessage = writable('');
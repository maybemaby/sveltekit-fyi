import type { Handle, HandleServerError } from '@sveltejs/kit';
import { dev } from '$app/environment';
import { initLogger } from 'evlog';
import { createEvlogHooks } from 'evlog/sveltekit';

initLogger({
	env: {
		environment: !dev ? 'production' : 'development',
		service: 'sveltekit-fyi',
		version: '0.1.0'
	},
	pretty: dev,
	sampling: {
		rates: {
			info: dev ? 100 : 50
		}
	}
});

const { handle: evlogHandle, handleError: evlogHandleError } = createEvlogHooks();

export const handleError: HandleServerError = async ({ error, event, message, status }) => {
	return evlogHandleError({ error, event, message, status });
};

export const handle: Handle = evlogHandle as Handle;

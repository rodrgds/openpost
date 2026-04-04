import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs));
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type WithoutChild<T> = T extends { child?: any } ? Omit<T, 'child'> : T;
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type WithoutChildren<T> = T extends { children?: any } ? Omit<T, 'children'> : T;
export type WithoutChildrenOrChild<T> = WithoutChildren<WithoutChild<T>>;
export type WithElementRef<T, U extends HTMLElement = HTMLElement> = T & { ref?: U | null };

export function getPlatformName(platform: string): string {
	switch (platform) {
		case 'x':
			return 'X (Twitter)';
		case 'mastodon':
			return 'Mastodon';
		case 'threads':
			return 'Threads';
		case 'bluesky':
			return 'Bluesky';
		case 'linkedin':
			return 'LinkedIn';
		default:
			return platform;
	}
}

export function getStatusColor(status: string): string {
	const colors: Record<string, string> = {
		draft: 'bg-muted text-muted-foreground',
		scheduled: 'bg-blue-500/10 text-blue-600 dark:text-blue-400',
		publishing: 'bg-yellow-500/10 text-yellow-600 dark:text-yellow-400',
		published: 'bg-green-500/10 text-green-600 dark:text-green-400',
		failed: 'bg-red-500/10 text-red-600 dark:text-red-400'
	};
	return colors[status] || 'bg-muted text-muted-foreground';
}

export function getPlatformColor(platform: string): string {
	const colors: Record<string, string> = {
		x: 'bg-black',
		mastodon: 'bg-indigo-500',
		threads: 'bg-orange-500',
		bluesky: 'bg-sky-500',
		linkedin: 'bg-blue-600'
	};
	return colors[platform] || 'bg-gray-500';
}

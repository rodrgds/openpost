import { clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';

/**
 * @param {import('clsx').ClassValue[]} inputs
 */
export function cn(...inputs) {
	return twMerge(clsx(inputs));
}

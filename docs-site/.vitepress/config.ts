import { defineConfig } from 'vitepress';

// Default to root-path hosting so custom-domain deployments work without extra config.
// Repository-path deployments (for example, GitHub Pages at /openpost/) should set OPENPOST_DOCS_BASE explicitly.
const docsBase = process.env.OPENPOST_DOCS_BASE?.trim() || '/';

const docsSidebar = [
	{
		text: 'Getting Started',
		collapsed: false,
		items: [
			{ text: 'What is OpenPost?', link: '/guide/what-is-openpost' },
			{ text: 'Quickstart', link: '/guide/quickstart' },
			{ text: 'Concepts', link: '/guide/concepts' },
		],
	},
	{
		text: 'Deployment',
		collapsed: false,
		items: [
			{ text: 'Docker Compose', link: '/installation/docker-compose' },
			{ text: 'Single Binary', link: '/installation/binary' },
			{ text: 'Nix Module', link: '/installation/nix-module' },
			{ text: 'Reverse Proxy', link: '/installation/reverse-proxy' },
		],
	},
	{
		text: 'Configuration',
		collapsed: false,
		items: [
			{ text: 'Environment Variables', link: '/configuration/environment-variables' },
			{ text: 'Database', link: '/configuration/database' },
			{ text: 'Media Storage', link: '/configuration/media-storage' },
			{ text: 'CORS and URLs', link: '/configuration/cors-and-urls' },
		],
	},
	{
		text: 'Providers',
		collapsed: false,
		items: [
			{ text: 'Overview', link: '/providers/overview' },
			{ text: 'X', link: '/providers/x' },
			{ text: 'Mastodon', link: '/providers/mastodon' },
			{ text: 'Bluesky', link: '/providers/bluesky' },
			{ text: 'LinkedIn', link: '/providers/linkedin' },
			{ text: 'Threads', link: '/providers/threads' },
		],
	},
	{
		text: 'Using OpenPost',
		collapsed: false,
		items: [
			{ text: 'Accounts', link: '/usage/accounts' },
			{ text: 'Composing Posts', link: '/usage/composing-posts' },
			{ text: 'Scheduling', link: '/usage/scheduling' },
			{ text: 'Media Library', link: '/usage/media-library' },
		],
	},
	{
		text: 'Operations',
		collapsed: false,
		items: [
			{ text: 'Backups', link: '/operations/backups' },
			{ text: 'Upgrades', link: '/operations/upgrades' },
			{ text: 'Troubleshooting', link: '/operations/troubleshooting' },
		],
	},
];

const developmentSidebar = [
	{
		text: 'Development',
		collapsed: false,
		items: [
			{ text: 'Setup', link: '/development/setup' },
			{ text: 'Architecture', link: '/development/architecture' },
			{ text: 'API Reference', link: '/development/api-reference' },
			{ text: 'Frontend', link: '/development/frontend' },
			{ text: 'Backend', link: '/development/backend' },
			{ text: 'Platform Adapters', link: '/development/platform-adapters' },
			{ text: 'Background Jobs', link: '/development/background-jobs' },
			{ text: 'Testing', link: '/development/testing' },
			{ text: 'Contributing', link: '/development/contributing' },
		],
	},
];

export default defineConfig({
	title: 'OpenPost',
	description: 'A lightweight, self-hosted social media scheduler.',
	base: docsBase,
	cleanUrls: true,
	lastUpdated: true,
	head: [
		['link', { rel: 'icon', href: `${docsBase}assets/brand/icon.svg` }],
		['meta', { property: 'og:type', content: 'website' }],
		['meta', { property: 'og:title', content: 'OpenPost' }],
		['meta', { property: 'og:description', content: 'A lightweight, self-hosted social media scheduler.' }],
		['meta', { property: 'og:image', content: `${docsBase}assets/brand/og-image.svg` }],
	],
	themeConfig: {
		logo: '/assets/brand/icon.svg',
		nav: [
			{ text: 'Guide', link: '/guide/quickstart' },
			{ text: 'Installation', link: '/installation/docker-compose' },
			{ text: 'Providers', link: '/providers/overview' },
			{ text: 'Operations', link: '/operations/backups' },
			{ text: 'Development', link: '/development/setup' },
		],
		socialLinks: [{ icon: 'github', link: 'https://github.com/rodrgds/openpost' }],
		search: {
			provider: 'local',
		},
		editLink: {
			pattern: 'https://github.com/rodrgds/openpost/edit/main/docs-site/:path',
			text: 'Edit this page on GitHub',
		},
		footer: {
			message: 'Released under the MIT License.',
			copyright: 'Copyright © Rodrigo Dias',
		},
		sidebar: {
			'/development/': developmentSidebar,
			'/': docsSidebar,
		},
	},
});

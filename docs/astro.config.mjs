// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

// Deployed as a GitHub Pages project site (https://manuelzzz.github.io/versiongate),
// not a custom domain — see specs/decisions/docs-site.md.
export default defineConfig({
	site: 'https://manuelzzz.github.io',
	base: '/versiongate',
	integrations: [
		starlight({
			title: 'VersionGate',
			description:
				'A self-hosted, API-first service for managing mobile application releases and evaluating update policies.',
			social: [
				{ icon: 'github', label: 'GitHub', href: 'https://github.com/manuelzzz/versiongate' },
			],
			editLink: {
				baseUrl: 'https://github.com/manuelzzz/versiongate/edit/main/docs/',
			},
			sidebar: [
				{ label: 'Installation', slug: 'installation' },
				{
					label: 'CLI',
					items: [
						{ label: 'Bootstrap', slug: 'bootstrap' },
						{ label: 'Managing the database schema', slug: 'migrations' },
					],
				},
				{
					label: 'API',
					items: [
						{ label: 'Publishing Releases', slug: 'publishing-releases' },
						{ label: 'Update Check', slug: 'update-check' },
					],
				},
			],
		}),
	],
});

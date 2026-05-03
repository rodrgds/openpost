import DefaultTheme from 'vitepress/theme';
import type { Theme } from 'vitepress';
import openapiSpec from '../../.generated/openapi.json' with { type: 'json' };
import { theme as openapiTheme, useOpenapi } from 'vitepress-openapi/client';
import 'vitepress-openapi/dist/style.css';
import './custom.css';

export default {
	extends: DefaultTheme,
	async enhanceApp(ctx) {
		useOpenapi({
			spec: openapiSpec,
			config: {
				spec: {
					groupByTags: true,
					showPathsSummary: true,
				},
			},
		});

		await DefaultTheme.enhanceApp?.(ctx);
		await openapiTheme.enhanceApp(ctx);
	},
} satisfies Theme;

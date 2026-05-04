import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

export default defineConfig({
  site: 'https://jvcorredor.github.io',
  base: '/bytheway',
  integrations: [
    starlight({
      title: 'bytheway',
      description:
        'A Go CLI for task management designed primarily for LLM-agent consumption (Claude Code), usable as a normal terminal CLI.',
      social: [
        { icon: 'github', label: 'GitHub', href: 'https://github.com/jvcorredor/bytheway' },
      ],
      editLink: {
        baseUrl: 'https://github.com/jvcorredor/bytheway/edit/main/docs/',
      },
      lastUpdated: true,
      sidebar: [
        {
          label: 'Getting started',
          items: [
            { label: 'Index', slug: 'index' },
            { label: 'Install', slug: 'install' },
            { label: 'CLI reference', slug: 'cli' },
          ],
        },
        {
          label: 'Reference',
          items: [
            { label: 'File & storage format', slug: 'storage' },
            { label: 'Configuration', slug: 'config' },
            { label: 'Agentic usage', slug: 'agentic' },
            { label: 'Releases', slug: 'releases' },
          ],
        },
        {
          label: 'Internals',
          items: [{ label: 'Design notes', slug: 'design' }],
        },
      ],
    }),
  ],
});

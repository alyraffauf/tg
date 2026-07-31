// @ts-check
import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";

// https://astro.build/config
export default defineConfig({
  integrations: [
    starlight({
      title: "tg",
      social: [
        {
          icon: "github",
          label: "Tangled",
          href: "https://tangled.org/aly.codes/tg",
        },
      ],
      components: {
        // Override SocialIcons to render the Tangled Dolly mark for the
        // tangled.org link instead of a generic icon.
        SocialIcons: "./src/components/SocialIcons.astro",
        // Override Head to avoid "tg | tg" on the homepage (page title == site title).
        Head: "./src/components/Head.astro",
        // Override Footer to link to aly.codes instead of Starlight credits.
        Footer: "./src/components/Footer.astro",
      },
      sidebar: [
        {
          label: "Cookbooks",
          items: [{ autogenerate: { directory: "cookbooks" } }],
        },
        {
          label: "Reference",
          // Generated at build time from `tg man --markdown` (see scripts/gen-commands.mjs).
          items: [{ autogenerate: { directory: "reference/commands" } }],
        },
      ],
    }),
  ],
});

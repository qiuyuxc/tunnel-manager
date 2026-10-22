# Documentation

Edit `content/<page>.json`. Each page contains `zh-CN` and `en` Markdown sections;
the routes and bilingual navigation are generated from one structure. The old root
and `en/` Markdown files are build output and must not be edited directly.

Run `npm run content:generate` after editing content. `npm run build` also generates
pages before VitePress checks links and builds local search. Keep translated text in
both sections; language-specific prose still needs translation and review.

Navigation and translated labels live in `.vitepress/navigation.json` and
`.vitepress/labels.json`. Add or reorder pages once in the navigation manifest.

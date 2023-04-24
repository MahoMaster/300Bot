import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import AutoImport from "unplugin-auto-import/vite";
import path from "path";
// https://vitejs.dev/config/
export default defineConfig({
	base: "./",
	plugins: [
		vue(),
		AutoImport({
			imports: ["vue", "vue-router", "vue-i18n", "@vueuse/head", "@vueuse/core"],
			dts: "src/auto-import.d.ts"
		})
	],
	resolve: {
		alias: {
			"@": path.resolve(__dirname, "src")
		}
	}
});

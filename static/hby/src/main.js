import { createApp } from "vue";
import "./style.css";
import App from "./App.vue";
import { GiftRain } from "@nutui/nutui-bingo";
import "@nutui/nutui-bingo/dist/style.css";

const app = createApp(App);
app.use(GiftRain);
// createApp(App)
app.mount("#app");

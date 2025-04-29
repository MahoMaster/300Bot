import { createApp } from "vue";
import "./style.css";
import App from "./App.vue";
import nutbig from "./dist/nutbig.es.js";
import "./dist/style.css";
import { GuessGift } from "./dist/nutbig.es.js";
// console.log(GuessGift);
const app = createApp(App);
// nutbig.install(app);
// console.log(nutbig);
app.use(GuessGift);
// createApp(App)
app.mount("#app");

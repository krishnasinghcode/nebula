import { createRoot } from "react-dom/client";
import App from "./App";
import "./index.css";

// Theme before first paint: default light; persisted choice wins.
const saved = localStorage.getItem("orbit-theme");
if (saved === "dark" || saved === "light") {
  document.documentElement.dataset.theme = saved;
} else {
  document.documentElement.dataset.theme = "light";
}

createRoot(document.getElementById("root")!).render(<App />);

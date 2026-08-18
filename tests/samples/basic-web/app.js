// Minimal SPA-router smoke so routing works without React/Vue.
const root = document.getElementById("root");
const route = document.getElementById("route");
const draw = (text) => {
  const c = document.getElementById("c");
  const ctx = c.getContext("2d");
  ctx.fillStyle = "#f7f7f9";
  ctx.fillRect(0, 0, 320, 240);
  ctx.fillStyle = "#111";
  ctx.font = "20px system-ui";
  ctx.fillText(text, 40, 120);
};

const render = () => {
  const path = location.pathname || "/";
  route.textContent = path;
  const label = path === "/" ? "w2e smoke"
    : path === "/about" ? "about"
    : path === "/dashboard" ? "dashboard"
    : path.replace("/", "") + " (route)";
  root.textContent = label;
  draw(label);
};

window.addEventListener("popstate", render);

document.querySelectorAll("nav a").forEach((a) => {
  a.addEventListener("click", (e) => {
    e.preventDefault();
    history.pushState({}, "", a.getAttribute("href"));
    render();
  });
});

render();

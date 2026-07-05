// Render one GitHub-style commit heatmap per year from /data/contributions.csv.
const CELL = 12, GAP = 2, WEEK = CELL + GAP, ROWS = 7;
const COLORS = ["#ebedf0", "#9be9a8", "#40c463", "#30a14e", "#216e39"];
const SVGNS = "http://www.w3.org/2000/svg";

const bucket = (n, max) => {
  if (n <= 0) return 0;
  if (max <= 4) return Math.min(n, 4);
  return Math.min(4, 1 + Math.floor((n - 1) / (max / 4)));
};

const dayOfWeek = (d) => (d.getUTCDay() + 6) % 7; // Mon=0 … Sun=6

function parseCSV(text) {
  const counts = new Map();
  for (const line of text.trim().split("\n").slice(1)) {
    const [date, count] = line.split(",");
    if (date) counts.set(date, Number(count) || 0);
  }
  return counts;
}

// yearGrid returns the SVG heatmap for a year and that year's commit total.
function yearGrid(year, counts) {
  const start = new Date(Date.UTC(year, 0, 1));
  const end = new Date(Date.UTC(year, 11, 31));
  const daysInYear = Math.round((Date.UTC(year + 1, 0, 1) - start) / 86400000);
  const weeks = Math.ceil((dayOfWeek(start) + daysInYear) / 7);

  let max = 1, total = 0;
  for (const [d, c] of counts) {
    if (d.startsWith(String(year))) { max = Math.max(max, c); total += c; }
  }

  const svg = document.createElementNS(SVGNS, "svg");
  svg.setAttribute("class", "heatmap");
  svg.setAttribute("width", weeks * WEEK - GAP);
  svg.setAttribute("height", ROWS * WEEK - GAP);
  svg.setAttribute("role", "img");
  svg.setAttribute("aria-label", `${total} commit contributions in ${year}`);

  const firstCol = new Date(Date.UTC(year, 0, 1 - dayOfWeek(start)));
  for (let d = new Date(firstCol); d <= end; d.setUTCDate(d.getUTCDate() + 1)) {
    if (d < start) continue;
    const key = d.toISOString().slice(0, 10);
    const col = Math.floor((d - firstCol) / 86400000 / 7);
    const row = dayOfWeek(d);
    const n = counts.get(key) || 0;
    const rect = document.createElementNS(SVGNS, "rect");
    rect.setAttribute("class", "cell");
    rect.setAttribute("x", col * WEEK);
    rect.setAttribute("y", row * WEEK);
    rect.setAttribute("width", CELL);
    rect.setAttribute("height", CELL);
    rect.setAttribute("rx", 2);
    rect.setAttribute("fill", COLORS[bucket(n, max)]);
    const title = document.createElementNS(SVGNS, "title");
    title.textContent = `${key}: ${n} commit${n === 1 ? "" : "s"}`;
    rect.appendChild(title);
    svg.appendChild(rect);
  }
  return { svg, total };
}

function renderLegend() {
  const scale = document.querySelector("#legend .scale");
  for (const c of COLORS) {
    const s = document.createElement("span");
    s.style.background = c;
    scale.appendChild(s);
  }
  document.getElementById("legend").hidden = false;
}

async function main() {
  const chart = document.getElementById("chart");
  try {
    const res = await fetch("/data/contributions.csv");
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const counts = parseCSV(await res.text());
    if (counts.size === 0) {
      chart.textContent = "No contribution data yet — the sync is still populating.";
      return;
    }
    const years = [...new Set([...counts.keys()].map((d) => +d.slice(0, 4)))]
      .sort((a, b) => b - a);
    chart.innerHTML = "";
    chart.classList.remove("empty");
    for (const y of years) {
      const { svg, total } = yearGrid(y, counts);
      const section = document.createElement("section");
      section.className = "year";
      const head = document.createElement("div");
      head.className = "year-head";
      head.innerHTML =
        `<h2>${y}</h2><span class="total">${total.toLocaleString()} commit${total === 1 ? "" : "s"}</span>`;
      const scroll = document.createElement("div");
      scroll.className = "cal-scroll";
      scroll.appendChild(svg);
      section.append(head, scroll);
      chart.appendChild(section);
    }
    renderLegend();
  } catch (e) {
    chart.textContent = "Failed to load contribution data.";
  }
}

main();

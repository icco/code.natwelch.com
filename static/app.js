// Fetch daily commit counts and render one GitHub-style heatmap per year.
const CELL = 12, GAP = 2, PAD = 24, WEEK = CELL + GAP;
const COLORS = ["#ebedf0", "#9be9a8", "#40c463", "#30a14e", "#216e39"];

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

function yearSVG(year, counts) {
  const start = new Date(Date.UTC(year, 0, 1));
  const end = new Date(Date.UTC(year, 11, 31));
  let max = 1;
  for (const [d, c] of counts) if (d.startsWith(String(year))) max = Math.max(max, c);

  const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  const weeks = 53;
  svg.setAttribute("width", PAD + weeks * WEEK);
  svg.setAttribute("height", PAD + 7 * WEEK + 8);
  svg.setAttribute("role", "img");
  svg.setAttribute("aria-label", `Commit contributions for ${year}`);

  const label = document.createElementNS(svg.namespaceURI, "text");
  label.setAttribute("x", PAD);
  label.setAttribute("y", 14);
  label.setAttribute("class", "yr");
  label.textContent = year;
  svg.appendChild(label);

  const firstCol = new Date(Date.UTC(year, 0, 1 - dayOfWeek(start)));
  for (let d = new Date(firstCol); d <= end; d.setUTCDate(d.getUTCDate() + 1)) {
    if (d < start) continue;
    const key = d.toISOString().slice(0, 10);
    const col = Math.floor((d - firstCol) / 86400000 / 7);
    const row = dayOfWeek(d);
    const n = counts.get(key) || 0;
    const rect = document.createElementNS(svg.namespaceURI, "rect");
    rect.setAttribute("class", "cell");
    rect.setAttribute("x", PAD + col * WEEK);
    rect.setAttribute("y", PAD - 8 + row * WEEK);
    rect.setAttribute("width", CELL);
    rect.setAttribute("height", CELL);
    rect.setAttribute("rx", 2);
    rect.setAttribute("fill", COLORS[bucket(n, max)]);
    const title = document.createElementNS(svg.namespaceURI, "title");
    title.textContent = `${key}: ${n} commit${n === 1 ? "" : "s"}`;
    rect.appendChild(title);
    svg.appendChild(rect);
  }
  return svg;
}

async function main() {
  const chart = document.getElementById("chart");
  try {
    const res = await fetch("/data/contributions.csv");
    const counts = parseCSV(await res.text());
    if (counts.size === 0) {
      chart.textContent = "No contribution data yet — the sync is still populating.";
      return;
    }
    const years = [...new Set([...counts.keys()].map((d) => Number(d.slice(0, 4))))].sort((a, b) => b - a);
    chart.innerHTML = "";
    for (const y of years) {
      const wrap = document.createElement("div");
      wrap.className = "mb3 overflow-x-auto";
      wrap.appendChild(yearSVG(y, counts));
      chart.appendChild(wrap);
    }
  } catch (e) {
    chart.textContent = "Failed to load contribution data.";
  }
}

main();

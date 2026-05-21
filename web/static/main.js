import { html, render } from 'https://unpkg.com/lighterhtml?module';

const icons = {
	default: '<svg xmlns="http://www.w3.org/2000/svg" xml:space="preserve" viewBox="31.4 0 449.1 512"><path d="m185.9 89.8 139.2 80.8v161.7l-139.2-80.8zm155.4 80.9v161.7l139.2-80.8V89.8zM31.4 0v161.7l139.2 80.8V80.8zm154.5 431.2L325.2 512V350.3L186 269.5v161.7z" style="fill-rule:evenodd;clip-rule:evenodd;fill:#a067da"/></svg>',
}

const cont = document.getElementById("graph");
const sidebar = document.getElementById("node-details-sidebar");
const graph = JSON.parse(cont.dataset.config);
const RESOURCE_CARD_WIDTH = 320;
const RESOURCE_CARD_HEIGHT = 96;
const stateColors = {
	create: "#27ae60",
	modify: "#f39c12",
	replace: "#8e44ad",
	delete: "#e74c3c",
	noop: "transparent",
};

function escapeSvgText(value) {
	return String(value ?? "")
		.replaceAll("&", "&amp;")
		.replaceAll("<", "&lt;")
		.replaceAll(">", "&gt;")
		.replaceAll('"', "&quot;")
		.replaceAll("'", "&#39;");
}

function resourceCardImage(node) {
	const name = escapeSvgText(node.name || node.address);
	const type = escapeSvgText(node.type || "");
	const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${RESOURCE_CARD_WIDTH}" height="${RESOURCE_CARD_HEIGHT}" viewBox="0 0 ${RESOURCE_CARD_WIDTH} ${RESOURCE_CARD_HEIGHT}">
<rect width="${RESOURCE_CARD_WIDTH}" height="${RESOURCE_CARD_HEIGHT}" fill="#fff" stroke="#d0d5dd"/><line x1="72" y1="0" x2="72" y2="${RESOURCE_CARD_HEIGHT}" stroke="#d0d5dd"/>
<line x1="72" y1="48" x2="${RESOURCE_CARD_WIDTH}" y2="48" stroke="#eaecf0"/>
<text x="92" y="34" font-family="-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif" font-size="17" font-weight="700" fill="#1d2939">${name}</text>
<text x="92" y="70" font-family="-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif" font-size="14" fill="#667085">${type}</text>
</svg>`;

	return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`;
}

const elements = [];
for (const [id, n] of Object.entries(graph.nodes)) {
	n.address = id;
	const state = n.state || "noop";
	elements.push({
		data: {
			id,
			label: id,
			kind: n.kind,
			name: n.name,
			type: n.type,
			state,
			stateColor: stateColors[state] || "transparent",
			cardImage: n.kind === "resource" ? resourceCardImage(n) : "",
		},
	});
}

const cy = cytoscape({
	container: document.getElementById("graph"),
	elements: elements,
	layout: { name: "breadthfirst", directed: true },
	style: [
		{
			selector: "node",
			style: { label: "data(label)", "background-color": "#ccc" }
		},
		{
			selector: 'node[kind="resource"]',
			style: {
				shape: "rectangle",
				width: RESOURCE_CARD_WIDTH,
				height: RESOURCE_CARD_HEIGHT,
				label: "",
				"background-image": "data(cardImage)",
				"background-fit": "none",
				"background-width": RESOURCE_CARD_WIDTH,
				"background-height": RESOURCE_CARD_HEIGHT,
				"background-opacity": 1,
				"background-color": "transparent",
				"underlay-padding": 2,
				"underlay-color": "data(stateColor)",
				"underlay-opacity": 0,
			}
		},
	],
});

let pulseOn = false;

setInterval(() => {
	pulseOn = !pulseOn;
	const nodes = cy.nodes('node[kind="resource"][state!="noop"]');

	cy.batch(() => {
		nodes.stop();
		nodes.animate({
			style: {
				"underlay-padding": pulseOn ? 3 : 1,
				"underlay-opacity": pulseOn ? 1 : 0.35,
			},
			duration: 700,
		});
	});
}, 800);

function renderNodeDetails(node) {
	if (!node) {
		closeNodeDetails();
		return;
	}

	const entries = [
		["Kind", node.kind],
		["Type", node.type],
		["Mode", node.mode],
		["State", node.state],
		["Parent", node.parent],
	].filter(([_, value]) => value);

	render(sidebar, html`
		<aside class="node-details-panel" aria-label="Node details">
			<header class="node-details-header">
				<div>
					<div class="node-details-eyebrow">Node</div>
					<h2>${node.address}</h2>
				</div>
				<button class="node-details-close" type="button" aria-label="Close node details" onclick=${closeNodeDetails}>&times;</button>
			</header>
			<div class="node-details-content">
				${entries.length ? html`
					<dl class="node-details-list">
						${entries.map(([label, value]) => html`
							<div>
								<dt>${label}</dt>
								<dd>${value}</dd>
							</div>
						`)}
					</dl>
				` : ""}
				${node.parameters?.length ? html`
					<section class="node-details-section">
						<h3>Parameters</h3>
						<ul>
							${node.parameters.map((parameter) => html`<li>${parameter}</li>`)}
						</ul>
					</section>
				` : ""}
				${node.changes ? html`
					<section class="node-details-section">
						<h3>Changes</h3>
						${node.changes.before ? html`<pre>${JSON.stringify(node.changes.before, null, 2)}</pre>` : ""}
						${node.changes.after ? html`<pre>${JSON.stringify(node.changes.after, null, 2)}</pre>` : ""}
					</section>
				` : ""}
			</div>
		</aside>
	`);
	sidebar.classList.add("is-open");
}

function closeNodeDetails() {
	render(sidebar, html``);
	sidebar.classList.remove("is-open");
}

cy.on("tap", "node", (event) => {
	const id = event.target.id();
	renderNodeDetails(graph.nodes[id]);
});

cy.on("tap", (event) => {
	if (event.target === cy) {
		closeNodeDetails();
	}
});

import { html, render } from 'https://unpkg.com/lighterhtml?module';

const icons = {
	default_resource: {
		viewBox: "31.4 0 449.1 512",
		body: `<path d="m185.9 89.8 139.2 80.8v161.7l-139.2-80.8zm155.4 80.9v161.7l139.2-80.8V89.8zM31.4 0v161.7l139.2 80.8V80.8zm154.5 431.2L325.2 512V350.3L186 269.5v161.7z" style="fill-rule:evenodd;clip-rule:evenodd;fill:#a067da"/>`
	},
	loop: {
		viewBox: "0 0 21 21",
		body: `
<path d="M0 0h21v21H0z" fill="none" />
<path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" d="m13.5 5.5l-2 10m-2-10l-2 10m-1-7h9m-10 4h9" />
`
	}
}

const sidebar = document.getElementById("node-details-sidebar");

const graph = JSON.parse(document.getElementById("initial-graph").text);
console.log(graph)
const RESOURCE_CARD_WIDTH = 340;
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
	const icon = icons[node.type] || icons.default_resource;
	const svg = `
<svg xmlns="http://www.w3.org/2000/svg" width="${RESOURCE_CARD_WIDTH}" height="${RESOURCE_CARD_HEIGHT}" viewBox="0 0 ${RESOURCE_CARD_WIDTH} ${RESOURCE_CARD_HEIGHT}">
<rect width="${RESOURCE_CARD_WIDTH}" height="${RESOURCE_CARD_HEIGHT}" fill="#fff" rx="8" ry="8" stroke="#d0d5dd"/>
<svg xmlns="http://www.w3.org/2000/svg" x="15" y="10" width="62" height="${RESOURCE_CARD_HEIGHT - 20}" viewBox="${icon.viewBox}">
  ${icon.body}
</svg>
<line x1="92" y1="0" x2="92" y2="${RESOURCE_CARD_HEIGHT}" stroke="#d0d5dd"/>
<line x1="92" y1="48" x2="${RESOURCE_CARD_WIDTH}" y2="48" stroke="#eaecf0"/>
<text x="102" y="34" font-family="-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif" font-size="17" font-weight="700" fill="#1d2939">${name}</text>
<text x="102" y="70" font-family="-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif" font-size="14" fill="#667085">${type}</text>
</svg>`;

	return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`;
}

function loopImage(node) {
	const name = escapeSvgText(node.name || node.address);
	const icon = icons.loop
	const svg = `
<svg xmlns="http://www.w3.org/2000/svg" width="${RESOURCE_CARD_WIDTH}" height="${RESOURCE_CARD_HEIGHT}" viewBox="0 0 ${RESOURCE_CARD_WIDTH} ${RESOURCE_CARD_HEIGHT}">
<rect x="0" y="0" width="${RESOURCE_CARD_WIDTH}" height="${RESOURCE_CARD_HEIGHT}" fill="#fff" stroke="#d0d5dd" stroke-width="5" />
<svg xmlns="http://www.w3.org/2000/svg" x="15" y="10" width="62" height="${RESOURCE_CARD_HEIGHT - 20}" viewBox="${icon.viewBox}">
  ${icon.body}
</svg>
<line x1="92" y1="0" x2="92" y2="${RESOURCE_CARD_HEIGHT}" stroke="#d0d5dd"/>
<text x="102" y="${(RESOURCE_CARD_HEIGHT + 8.5) / 2}" font-family="-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif" font-size="17" font-weight="700" fill="#1d2939">${name}</text>

</svg>`;
	return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`;
}

function getImage(node) {
	switch(node.kind) {
		case "resource": return resourceCardImage(node);
		case "resource_loop": return loopImage(node);
	}
}

const nodes = [];
for (const [id, n] of Object.entries(graph.nodes)) {
	const state = n.state || "noop";
	nodes.push({
		data: {
			id: id,
			label: id,
			kind: n.kind,
			name: n.name,
			type: n.type,
			parent: n.parent,
			state: state,
			stateColor: stateColors[state] || "transparent",
			cardImage: getImage(n),
		},
	});
}

const edges = [];
for (const edge of graph.edges) {
	edges.push({
		data: {
			id: `${edge.from}>${edge.to}`,
			source: edge.from,
			target: edge.to,
			type: edge.type
		},
	})
}

const cy = cytoscape({
	container: document.getElementById("graph"),
	elements: {
		nodes: nodes,
		edges: edges,
	},
	layout: { name: "breadthfirst", directed: true },
	style: [
		{
			selector: "node",
			style: {
				label: "data(label)",
				shape: "rectangle",
				"background-color": "#ccc"
			}
		},
		{
			selector: 'node[kind="resource"]',
			style: {
				width: RESOURCE_CARD_WIDTH,
				height: RESOURCE_CARD_HEIGHT,
				label: "",
				"background-image": "data(cardImage)",
				"background-width": RESOURCE_CARD_WIDTH,
				"background-height": RESOURCE_CARD_HEIGHT,
				"background-opacity": 0,
				"background-color": "transparent",
				"underlay-padding": 0,
				"underlay-color": "data(stateColor)",
				"underlay-opacity": 0,
			}
		},
		{
			selector: 'node[kind="resource_loop"]',
			style: {
				"label": "",
				"background-opacity": 0.4,
				"border-width": 2,
				"border-color": "data(stateColor)",
				"border-style": "dashed",
				"padding": "24",
				"background-image": "data(cardImage)",
				"background-fit": "none",
				"background-width": RESOURCE_CARD_WIDTH,
				"background-position-x": "0px",
				"background-height": RESOURCE_CARD_HEIGHT,
				"background-position-y": `-${RESOURCE_CARD_HEIGHT}px`,
				"background-clip": "none",
			},
		},
		{
			selector: 'node[kind="resource_loop"][collapsed]',
			style: {
				width: RESOURCE_CARD_WIDTH - 48
			}
		},
		{
			selector: "edge",
			style: {
				"line-color": "#98a2b3",
				"target-arrow-color": "#98a2b3",
				"target-arrow-shape": "triangle",
				"curve-style": "bezier",
			},
		},
		{
			selector: 'edge[type="transitive"]',
			style: {
				// display: "none",
				"line-style": "dashed",
				"opacity": 0.25,
			},
		},
		{
			selector: 'edge[type="instance"]',
			style: {
				display: "none",
			},
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
				"underlay-opacity": pulseOn ? 0.9 : 0.35,
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

cy.on("dbltap", 'node[kind="resource_loop"]', (event) => {
	const parent = event.target;
	const collapsed = parent.data("collapsed") === true;
	const descendants = parent.descendants();
	const descendantEdges = descendants.connectedEdges();

	if (collapsed) {
		descendants.show();
		descendantEdges.show();
		parent.data("collapsed", false);
	} else {
		descendants.hide();
		descendantEdges.hide();
		parent.data("collapsed", true);
	}
});


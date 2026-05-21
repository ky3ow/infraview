import { html, render } from 'https://unpkg.com/lighterhtml?module';

const icons = {
	module: {
		viewBox: "31.4 0 449.1 512",
		body: `<path d="m185.9 89.8 139.2 80.8v161.7l-139.2-80.8zm155.4 80.9v161.7l139.2-80.8V89.8zM31.4 0v161.7l139.2 80.8V80.8zm154.5 431.2L325.2 512V350.3L186 269.5v161.7z" style="fill-rule:evenodd;clip-rule:evenodd;fill:#a067da"/>`
	},
	default_resource: {
		viewBox: "0 0 256 256",
		body: `
<path d="M0 0h256v256H0z" fill="none" />
<path fill="currentColor" d="m222.72 67.9l-88-48.17a13.9 13.9 0 0 0-13.44 0l-88 48.18A14 14 0 0 0 26 80.18v95.64a14 14 0 0 0 7.28 12.27l88 48.18a13.92 13.92 0 0 0 13.44 0l88-48.18a14 14 0 0 0 7.28-12.27V80.18a14 14 0 0 0-7.28-12.28M127 30.25a2 2 0 0 1 1.92 0L212.51 76L128 122.24L43.49 76ZM39 177.57a2 2 0 0 1-1-1.75V86.66l84 46V223Zm177.92 0L134 223v-90.36l84-46v89.16a2 2 0 0 1-1 1.77Z" />
`
	},
	resource_loop: {
		viewBox: "0 0 21 21",
		body: `
<path d="M0 0h21v21H0z" fill="none" />
<path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" d="m13.5 5.5l-2 10m-2-10l-2 10m-1-7h9m-10 4h9" />
`
	},
	variable: {
		viewBox: "0 0 20 20",
		body: `
  <path d="M0 0h20v20H0z" fill="none" />
  <path fill="currentColor" d="M5.5 3A2.5 2.5 0 0 0 3 5.5v2.882c0 .493-.277.943-.715 1.167a.5.5 0 0 0 0 .902c.438.223.715.674.715 1.167V14.5A2.5 2.5 0 0 0 5.5 17a.5.5 0 0 0 0-1A1.5 1.5 0 0 1 4 14.5v-2.882c0-.615-.244-1.193-.662-1.618A2.3 2.3 0 0 0 4 8.382V5.5A1.5 1.5 0 0 1 5.5 4a.5.5 0 0 0 0-1m9 0A2.5 2.5 0 0 1 17 5.5v2.882c0 .493.277.943.715 1.167a.5.5 0 0 1 0 .902A1.31 1.31 0 0 0 17 11.618V14.5a2.5 2.5 0 0 1-2.5 2.5a.5.5 0 0 1 0-1a1.5 1.5 0 0 0 1.5-1.5v-2.882c0-.615.244-1.193.662-1.618A2.3 2.3 0 0 1 16 8.382V5.5A1.5 1.5 0 0 0 14.5 4a.5.5 0 0 1 0-1M7.907 6.21a.5.5 0 0 0-.814.58L9.386 10l-2.293 3.21a.5.5 0 0 0 .814.58L10 10.86l2.093 2.93a.5.5 0 0 0 .814-.58L10.614 10l2.293-3.21a.5.5 0 1 0-.814-.58L10 9.14z" />
`
	},
	output: {
		viewBox: "0 0 24 24",
		body: `
  <path d="M0 0h24v24H0z" fill="none" />
  <path fill="currentColor" d="M5.616 20q-.691 0-1.153-.462T4 18.384V5.616q0-.691.463-1.153T5.616 4h12.769q.69 0 1.153.463T20 5.616V6.5q0 .214-.143.357T19.5 7t-.357-.143T19 6.5v-.885q0-.23-.192-.423T18.384 5H5.616q-.231 0-.424.192T5 5.616v12.769q0 .23.192.423t.423.192h12.77q.23 0 .423-.192t.192-.423V17.5q0-.213.143-.357T19.5 17t.357.143t.143.357v.885q0 .69-.462 1.152T18.384 20zm13.463-7.5H9.885q-.214 0-.357-.143q-.144-.143-.144-.357t.144-.357t.357-.143h9.194l-2.721-2.765q-.14-.14-.144-.332q-.002-.191.144-.337q.14-.141.344-.141t.344.14l3.388 3.37q.131.13.184.267t.053.298t-.053.298t-.184.268l-3.388 3.388q-.14.14-.341.133t-.347-.153q-.14-.14-.14-.334t.14-.335z" />
`
	},
	readonly: {
		viewBox: "0 0 24 24",
		body: `
  <path d="M0 0h24v24H0z" fill="none" />
  <path fill="currentColor" d="M6 22q-.825 0-1.412-.587T4 20V10q0-.825.588-1.412T6 8h1V6q0-2.075 1.463-3.537T12 1t3.538 1.463T17 6v2h1q.825 0 1.413.588T20 10v10q0 .825-.587 1.413T18 22zm0-2h12V10H6zm7.413-3.588Q14 15.826 14 15t-.587-1.412T12 13t-1.412.588T10 15t.588 1.413T12 17t1.413-.587M9 8h6V6q0-1.25-.875-2.125T12 3t-2.125.875T9 6zM6 20V10z" />
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
	const readonlyIcon = icons.readonly;
	const svg = `
<svg xmlns="http://www.w3.org/2000/svg" width="${RESOURCE_CARD_WIDTH}" height="${RESOURCE_CARD_HEIGHT}" viewBox="0 0 ${RESOURCE_CARD_WIDTH} ${RESOURCE_CARD_HEIGHT}">
<rect width="${RESOURCE_CARD_WIDTH}" height="${RESOURCE_CARD_HEIGHT}" fill="#fff" rx="8" ry="8" stroke="#d0d5dd"/>
<svg xmlns="http://www.w3.org/2000/svg" x="15" y="10" width="62" height="${RESOURCE_CARD_HEIGHT - 20}" viewBox="${icon.viewBox}">
  ${icon.body}
</svg>
${node.mode == "data" ? `<svg xmlns="http://www.w3.org/2000/svg" x="62" y="30" width="30" heigth="30" viewBox="${readonlyIcon.viewBox}">
${readonlyIcon.body}
</svg>` : ""}
<line x1="92" y1="0" x2="92" y2="${RESOURCE_CARD_HEIGHT}" stroke="#d0d5dd"/>
<line x1="92" y1="48" x2="${RESOURCE_CARD_WIDTH}" y2="48" stroke="#eaecf0"/>
<text x="102" y="34" font-family="-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif" font-size="17" font-weight="700" fill="#1d2939">${name}</text>
<text x="102" y="70" font-family="-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif" font-size="14" fill="#667085">${type}</text>
</svg>`;

	return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`;
}

function standardImage(node) {
	const name = escapeSvgText(node.name || node.address);
	const icon = icons[node.kind] || icons.default_resource
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
		case "resource_loop":
		case "module":
		case "variable":
		case "output":
			return standardImage(node);
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
			mode: n.mode,
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
				shape: "round-rectangle",
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
			selector: 'node[mode="data"]',
			style: {
				"border-color": "#00b4d8",
				"border-cap": "round",
				"border-opacity": 1,
				"border-width": 3,
			}
		},
		{
			selector: 'node[kind="variable"],node[kind="output"]',
			style: {
				width: RESOURCE_CARD_WIDTH,
				height: RESOURCE_CARD_HEIGHT,
				label: "",
				"background-image": "data(cardImage)",
				"background-width": RESOURCE_CARD_WIDTH,
				"background-height": RESOURCE_CARD_HEIGHT,
				"background-opacity": 0,
				"background-color": "transparent",
			}
		},
		{
			selector: 'node[kind="resource_loop"], node[kind="module"]',
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
			selector: 'node[kind="resource_loop"][collapsed], node[kind="module"][collapsed]',
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
			selector: 'edge[type="summary"]',
			style: {
				// display: "none",
				"line-style": "dashed",
				"opacity": 1,
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

cy.on("dbltap", 'node[kind="resource_loop"], node[kind="module"]', (event) => {
	const parent = event.target;

	if (parent.data("collapsed")) {
		expandParent(parent)
	} else {
		collapseParent(parent);
	}
});

function summaryEdgeId(parentId, sourceId, targetId) {
	return `summary:${parentId}:${sourceId}>${targetId}`;
}

function collapseParent(parent) {
	const descendants = parent.descendants();
	const descendantIds = new Set(descendants.map(n => n.id()));
	const parentId = parent.id();

	const externalEdges = descendants.connectedEdges().filter(edge => {
		const sourceInside = descendantIds.has(edge.source().id());
		const targetInside = descendantIds.has(edge.target().id());

		return sourceInside !== targetInside;
	});

	const summaryEdges = externalEdges.map(edge => {
		const sourceInside = descendantIds.has(edge.source().id());
		const outsideNode = sourceInside ? edge.target() : edge.source();

		return {
			group: "edges",
			data: {
				id: summaryEdgeId(parentId, edge.source().id(), edge.target().id()),
				source: sourceInside ? parentId : outsideNode.id(),
				target: sourceInside ? outsideNode.id() : parentId,
				type: "summary",
				originalEdge: edge.id(),
			},
			classes: "summary-edge",
		};
	});

	cy.add(summaryEdges);

	descendants.hide();
	parent.data("collapsed", true);
}

function expandParent(parent) {
	const parentId = parent.id();

	cy.edges(`[type = "summary"]`).filter(edge => {
		return edge.data("source") === parentId || edge.data("target") === parentId;
	}).remove();

	parent.descendants().show();
	parent.data("collapsed", false);
}

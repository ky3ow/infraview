import { html, render } from 'https://unpkg.com/lighterhtml?module';
import icons from "./icons_default.js";

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

// States that participate in filtering.
const FILTERABLE_STATES = ["create", "delete", "modify", "noop", "replace"];

const stateLabels = {
	create: "Create",
	delete: "Delete",
	modify: "Modify",
	noop: "No-op",
	replace: "Replace",
};

const activeFilters = Object.fromEntries(FILTERABLE_STATES.map(s => [s, true]));

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
	wheelSensitivity: 0.3,
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
			selector: ".collapsed, .filtered",
			style: {
				display: "none"
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
				"underlay-opacity": 0.8,
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
				"padding-top": "96",
				"min-width": RESOURCE_CARD_WIDTH - 48,
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
			selector: "edge",
			style: {
				"line-color": "#98a2b3",
				"target-arrow-color": "#98a2b3",
				"target-arrow-shape": "triangle",
				"curve-style": "bezier",
			},
		},
		{
			selector: 'edge[type="transitive"], edge[type="instance"]',
			style: {
				display: "none",
				"line-style": "dashed",
				"opacity": 0.65,
			},
		},
		{
			selector: 'edge.shown',
			style: {
				display: "element",
			},
		},
		{
			selector: '.dimmed',
			style: {
				opacity: 0.2,
				"underlay-opacity": 0,
			},
		},
		{
			selector: 'edge[type="summary"]',
			style: {
				"line-style": "dashed",
				"opacity": 1,
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
			},
			duration: 700,
		});
	});
}, 800);

function renderChangeRows(changes) {
	if (!changes) return "";

	const before = changes.before;
	const after = changes.after;

	const isNew = !before;
	const isDeleted = !after;

	const beforeMap = before || {};
	const afterMap = after || {};

	const allKeys = new Map();
	for (const k of Object.keys(beforeMap)) allKeys.set(k, true);
	for (const k of Object.keys(afterMap)) allKeys.set(k, true);

	const rows = [];
	for (const key of allKeys.keys()) {
		const oldVal = beforeMap[key];
		const newVal = afterMap[key];

		if (oldVal == null && newVal == null) continue;

		let oldClass, oldContent, newClass, newContent;

		if (isNew) {
			oldClass = "value-null";
			oldContent = "<new>";
			newClass = "value-created";
			newContent = renderValueInline(newVal);
		} else if (isDeleted) {
			oldClass = "value-regular";
			oldContent = renderValueInline(oldVal);
			newClass = "value-deleted";
			newContent = "<deleted>";
		} else {
			oldClass = "value-regular";
			oldContent = renderValueInline(oldVal);

			if (newVal == null && oldVal != null) {
				newClass = "value-deleted";
				newContent = renderValueInline(newVal);
			} else if (newVal != null && oldVal == null) {
				newClass = "value-created";
				newContent = renderValueInline(newVal);
			} else if (JSON.stringify(oldVal) === JSON.stringify(newVal)) {
				newClass = "value-regular";
				newContent = renderValueInline(newVal);
			} else {
				newClass = "value-updated";
				newContent = renderValueInline(newVal);
			}
		}

		rows.push(html`
			<div class="change-row">
				<dt class="change-key">${key}</dt>
				<dd><span class=${oldClass}>${oldContent}</span></dd>
				<span class="change-arrow">→</span>
				<dd><span class=${newClass}>${newContent}</span></dd>
			</div>
		`);
	}

	if (rows.length === 0) return "";

	return html`
		<dl class="changes-table">
			${rows}
		</dl>
	`;
}

function renderValueInline(v) {
	if (typeof v === "object" && v !== null) {
		return JSON.stringify(v);
	}
	return String(v);
}

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
				${node.parameters?.length ? "": "" /* html`
					<section class="node-details-section">
						<h3>Parameters</h3>
						<ul>
							${node.parameters.map((parameter) => html`<li>${parameter}</li>`)}
						</ul>
					</section>
				` : ""*/}
				${node.changes ? html`
					<section class="node-details-section">
						<h3>Changes</h3>
						${renderChangeRows(node.changes)}
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

function countNodesByState() {
	const counts = {};
	for (const state of FILTERABLE_STATES) {
		counts[state] = cy.nodes(`[state="${state}"]`).length;
	}
	return counts;
}

function renderStateFilters() {
	const counts = countNodesByState();
	const filterContainer = document.getElementById("state-filters");

	// TODO move this hack to markup
	render(filterContainer, html`
		<div class="state-filter-list">
			${FILTERABLE_STATES.map(state => html`
				<label class="state-filter-item">
					<span class="state-filter-color-dot" style="background:${stateColors[state]};border:1px solid #ccc"></span>
					<span class="state-filter-label">${stateLabels[state]}</span>
					<span class="state-filter-count">${counts[state]}</span>
					<input
						type="checkbox"
						class="state-filter-checkbox"
						checked=${activeFilters[state]}
						onchange=${() => {
							activeFilters[state] = !activeFilters[state];
							applyFilters();
							renderStateFilters();
						}}
					/>
				</label>
			`)}
		</div>
		<button class="state-filter-reset-btn" onclick=${() => {
			for (const s of FILTERABLE_STATES) activeFilters[s] = true;
			applyFilters();
			renderStateFilters();
		}}>Show All</button>
	`);
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

// dim everything except containers when hovering deleted nodes or instances
cy.on("mouseover", "node", (event) => {
	const node = event.target;

	const isDeleted = node.data("state") === "delete";
	const parentId = node.data("parent");
	const parent = parentId ? cy.getElementById(parentId) : null;
	const isInstance = parent && parent.length > 0 && parent.data("kind") === "resource_loop";

	if (node.data("kind") == "resource_loop" || node.data("kind") == "module") {
		return;
	}

	if (!isDeleted && !isInstance) return;

	const shown = node.connectedEdges();
	shown.addClass("shown");

	const directlyConnected = shown.connectedNodes().union(node);

	cy.nodes().filter(n => (n.data("kind") !== "module") && (n.data("kind") !== "resource_loop") ).filter(n => !directlyConnected.has(n)).addClass("dimmed");
	cy.edges().filter(e => !shown.has(e)).addClass("dimmed");
});

cy.on("mouseout", "node", (event) => {
	cy.elements(".dimmed").removeClass("dimmed");
	cy.edges("edge.shown").removeClass("shown");
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

	descendants.addClass("collapsed");
	parent.data("collapsed", true);
}

function expandParent(parent) {
	const parentId = parent.id();

	cy.edges(`[type = "summary"]`).filter(edge => {
		return edge.data("source") === parentId || edge.data("target") === parentId;
	}).remove();

	parent.descendants().removeClass("collapsed");
	parent.data("collapsed", false);
}

function applyFilters() {
	const CONTAINER_KINDS = new Set(["module", "resource_loop", "module_loop"]);

	const ownHidden = new Set();
	for (const [state, visible] of Object.entries(activeFilters)) {
		if (!visible) {
			cy.nodes(`[state="${state}"]`).forEach(n => {
				// Modules/loops with modify state are exempt — never hidden
				if (state === "modify" && CONTAINER_KINDS.has(n.data("kind"))) return;
				ownHidden.add(n.id());
			});
		}
	}

	const cascadeHidden = (nodeId) => {
		cy.getElementById(nodeId).descendants().forEach(child => {
			const isExempt = child.data("state") === "modify" && CONTAINER_KINDS.has(child.data("kind"));
			if (isExempt) return;
			if (ownHidden.has(child.id())) return;
			ownHidden.add(child.id());
			cascadeHidden(child.id());
		});
	};

	for (const id of [...ownHidden]) {
		cascadeHidden(id);
	}

	const shouldBeVisible = new Set();
	for (const id of ownHidden) {
		const node = cy.getElementById(id);
		const visibleDescendants = node.descendants().some(d => !ownHidden.has(d.id()));
		if (visibleDescendants) {
			shouldBeVisible.add(id);
		}
	}

	cy.batch(() => {
		for (const id of ownHidden) {
			const node = cy.getElementById(id);
			if (shouldBeVisible.has(id)) {
				node.removeClass("filtered");
			} else {
				node.addClass("filtered");
			}
		}
		cy.nodes().forEach(n => {
			if (!ownHidden.has(n.id())) {
				n.removeClass("filtered");
			}
		});
	});
}

// Initialize filter UI on load
renderStateFilters();
applyFilters();

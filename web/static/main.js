import { html, render } from 'https://unpkg.com/lighterhtml?module';
import dagre from 'https://cdn.jsdelivr.net/npm/cytoscape-dagre@3.0.0/+esm';
import icons from "./icons_default.js";

cytoscape.use(dagre);

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

let state = {};

function init() {
	state.graph = JSON.parse(document.getElementById("graph-data").text);
	state.sidebar = document.getElementById("node-details-sidebar");
	state.activeFilters = Object.fromEntries(FILTERABLE_STATES.map(s => [s, true]));

	const [nodes, edges] = prepareNodes(state.graph);

	state.cy = cytoscape({
		container: document.getElementById("graph"),
		wheelSensitivity: 0.3,
		elements: {
			nodes: nodes,
			edges: edges,
		},
		layout: { name: "dagre", nodeSep: 200, rankSep: 150, spacingFactor: 1.2, rankDir: "LR"},
		style: getGraphStyles(),
	});

	state.pulseInterval = createPulsing();

	state.cy.on("tap", "node", cyHandleDetailsOpen);
	state.cy.on("tap", cyHandleDetailsClose);
	state.cy.on("dbltap", 'node[kind="resource_loop"], node[kind="module"]', cyHandleCollapsing);
	state.cy.on("mouseover", "node", cyHandleDimmingStart);
	state.cy.on("mouseout", "node", cyHandleDimmingEnd);

	renderStateFilters();
	applyFilters();
}

function cleanup() {
	if (!state.cy) return;
	if (!state.cy) return;

	state.cy.destroy();
	state.cy = null;

	if (state.pulseInterval) {
		clearInterval(state.pulseInterval);
		state.pulseInterval = null;
	}

	closeNodeDetails();
	state.graph = null;
	state.activeFilters = null;
}

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
	// TODO svg helper would be worth it
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

function prepareNodes(graph) {
	const nodes = [];
	for (const [id, n] of Object.entries(graph.nodes)) {
		const nodeState = n.state || "noop";
		nodes.push({
			data: {
				id: id,
				label: id,
				kind: n.kind,
				name: n.name,
				type: n.type,
				parent: n.parent,
				mode: n.mode,
				state: nodeState,
				stateColor: stateColors[nodeState] || "transparent",
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

	return [nodes, edges];
}

function createPulsing() {
	state.pulseOn = false;
	const timer = setInterval(() => {
		state.pulseOn = !state.pulseOn;
		const nodes = state.cy.nodes('node[kind="resource"][state!="noop"]');

		state.cy.batch(() => {
			nodes.stop();
			nodes.animate({
				style: {
					"underlay-padding": state.pulseOn ? 3 : 1,
				},
				duration: 700,
			});
		});
	}, 800);
}

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

	render(state.sidebar, html`
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
	state.sidebar.classList.add("is-open");
}

function closeNodeDetails() {
	render(state.sidebar, html``);
	state.sidebar.classList.remove("is-open");
}

function countNodesByState() {
	const counts = {};
	for (const filterState of FILTERABLE_STATES) {
		counts[filterState] = state.cy.nodes(`[state="${filterState}"]`).length;
	}
	return counts;
}

function renderStateFilters() {
	const counts = countNodesByState();
	const filterContainer = document.getElementById("state-filters");

	// TODO move this hack to markup
	render(filterContainer, html`
		<div class="state-filter-list">
			${FILTERABLE_STATES.map(filterState => html`
				<label class="state-filter-item">
					<span class="state-filter-color-dot" style="background:${stateColors[filterState]};border:1px solid #ccc"></span>
					<span class="state-filter-label">${stateLabels[filterState]}</span>
					<span class="state-filter-count">${counts[filterState]}</span>
					<input
						type="checkbox"
						class="state-filter-checkbox"
						checked=${state.activeFilters[filterState]}
						onchange=${() => {
							state.activeFilters[filterState] = !state.activeFilters[filterState];
							applyFilters();
							renderStateFilters();
						}}
					/>
				</label>
			`)}
		</div>
		<button class="state-filter-reset-btn" onclick=${() => {
			for (const s of FILTERABLE_STATES) state.activeFilters[s] = true;
			applyFilters();
			renderStateFilters();
		}}>Show All</button>
	`);
}

function cyHandleDetailsOpen(event) {
	const nodeKind = event.target.data("kind");
	if (nodeKind === "resource_loop" || nodeKind === "module") {
		return;
	}
	const id = event.target.id();
	renderNodeDetails(state.graph.nodes[id]);
}

function cyHandleDetailsClose(event) {
	if (event.target === state.cy) {
		closeNodeDetails();
	}
}

function cyHandleCollapsing(event) {
	const parent = event.target;

	if (parent.data("collapsed")) {
		expandParent(parent)
	} else {
		collapseParent(parent);
	}
}

function cyHandleDimmingStart(event) {
	const node = event.target;

	const isDeleted = node.data("state") === "delete";
	const parentId = node.data("parent");
	const parent = parentId ? state.cy.getElementById(parentId) : null;
	const isInstance = parent && parent.length > 0 && parent.data("kind") === "resource_loop";

	if (node.data("kind") == "resource_loop" || node.data("kind") == "module") {
		return;
	}

	if (!isDeleted && !isInstance) return;

	const shown = node.connectedEdges();
	shown.addClass("shown");

	const directlyConnected = shown.connectedNodes().union(node);

	state.cy.nodes().filter(n => (n.data("kind") !== "module") && (n.data("kind") !== "resource_loop") ).filter(n => !directlyConnected.has(n)).addClass("dimmed");
	state.cy.edges().filter(e => !shown.has(e)).addClass("dimmed");
}

function cyHandleDimmingEnd(event) {
	state.cy.elements(".dimmed").removeClass("dimmed");
	state.cy.edges("edge.shown").removeClass("shown");
}

function summaryEdgeId(parentId, sourceId, targetId) {
	return `summary:${parentId}:${sourceId}>${targetId}`;
}

function collapseParent(parent) {
	const descendants = parent.descendants();
	const descendantIds = new Set(descendants.map(n => n.id()));
	const parentId = parent.id();

	// What outside nodes is the parent already directly connected to?
	const parentOutsideTargets = new Set();
	parent.connectedEdges().forEach(e => {
		const srcId = e.source().id();
		const tgtId = e.target().id();
		if (srcId === parentId && !descendantIds.has(tgtId)) parentOutsideTargets.add(tgtId);
		if (tgtId === parentId && !descendantIds.has(srcId)) parentOutsideTargets.add(srcId);
	});

	// Collect all edges that cross the container boundary
	const externalEdges = descendants.connectedEdges().filter(edge => {
		const sourceInside = descendantIds.has(edge.source().id());
		const targetInside = descendantIds.has(edge.target().id());
		return sourceInside !== targetInside;
	});

	// Deduplicate by outside node, skip if parent already has a direct edge
	const seenOutside = new Set();
	const summaryEdges = [];

	for (const edge of externalEdges) {
		const sourceInside = descendantIds.has(edge.source().id());
		const outsideNode = sourceInside ? edge.target() : edge.source();
		const outsideId = outsideNode.id();

		const key = (sourceInside ? "out:" : "in:") + outsideId;
		if (seenOutside.has(key)) continue;
		if (parentOutsideTargets.has(outsideId)) continue;  // parent already covers it
		seenOutside.add(key);

		summaryEdges.push({
			group: "edges",
			data: {
				id: summaryEdgeId(parentId, edge.source().id(), edge.target().id()),
				source: sourceInside ? parentId : outsideId,
				target: sourceInside ? outsideId : parentId,
				type: "summary",
				originalEdge: edge.id(),
			},
			classes: "summary-edge",
		});
	}

	state.cy.add(summaryEdges);

	descendants.addClass("collapsed");
	parent.data("collapsed", true);
}

function expandParent(parent) {
	const parentId = parent.id();

	state.cy.edges(`[type = "summary"]`).filter(edge => {
		return edge.data("source") === parentId || edge.data("target") === parentId;
	}).remove();

	parent.descendants().removeClass("collapsed");
	parent.data("collapsed", false);
}

function applyFilters() {
	const CONTAINER_KINDS = new Set(["module", "resource_loop", "module_loop"]);

	const ownHidden = new Set();
	for (const [filterState, visible] of Object.entries(state.activeFilters)) {
		if (!visible) {
			state.cy.nodes(`[state="${filterState}"]`).forEach(n => {
				// Modules/loops with modify state are exempt — never hidden
				if (filterState === "modify" && CONTAINER_KINDS.has(n.data("kind"))) return;
				ownHidden.add(n.id());
			});
		}
	}

	const cascadeHidden = (nodeId) => {
		state.cy.getElementById(nodeId).descendants().forEach(child => {
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
		const node = state.cy.getElementById(id);
		const visibleDescendants = node.descendants().some(d => !ownHidden.has(d.id()));
		if (visibleDescendants) {
			shouldBeVisible.add(id);
		}
	}

	state.cy.batch(() => {
		for (const id of ownHidden) {
			const node = state.cy.getElementById(id);
			if (shouldBeVisible.has(id)) {
				node.removeClass("filtered");
			} else {
				node.addClass("filtered");
			}
		}
		state.cy.nodes().forEach(n => {
			if (!ownHidden.has(n.id())) {
				n.removeClass("filtered");
			}
		});
	});
}

function getGraphStyles() {
	return [
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
	]
}

document.addEventListener("htmx:afterSwap", (e) => {
	if (e.target.id === "graph-content") {
		cleanup();
		init();
	}
});

init();

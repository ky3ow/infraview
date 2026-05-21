import { html, render } from 'https://unpkg.com/lighterhtml?module';

const cont = document.getElementById("graph");
const sidebar = document.getElementById("node-details-sidebar");
const graph = JSON.parse(cont.dataset.config);

const elements = [];
for (const [id, n] of Object.entries(graph.nodes)) {
	n.address = id;
	elements.push({
		data: { id, label: id, action: n.changes?.actions?.[0] || "no-op" },
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
			selector: 'node[action="create"]',
			style: { "background-color": "#27ae60" }
		},
		{
			selector: 'node[action="delete"]',
			style: { "background-color": "#e74c3c" }
		},
		{
			selector: 'node[action="update"]',
			style: { "background-color": "#f39c12" }
		},
	],
});

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

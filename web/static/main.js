import { html, render } from 'https://unpkg.com/lighterhtml?module';

const cont = document.getElementById("graph");
const graph = JSON.parse(cont.dataset.config);
console.log(graph)

window.Graph = graph

const elements = [];
for (const [id, n] of Object.entries(graph.nodes)) {
	n.address = id;
	elements.push({
		data: { id, label: id, action: n.changes?.actions?.[0] || "no-op" },
	});
}
for (const e of graph.edges) {
	if (((e.from.split('.').length - 1) % 2 == 1) && ((e.to.split('.').length - 1) % 2 == 1) && (!e.from.startsWith('var.'))) {
		elements.push({ data: { source: e.from, target: e.to } });
	}
}

const cy = cytoscape({
	container: document.getElementById("graph"),
	elements,
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

let activeTab = 'parameters';
const popupEl = document.getElementById('popup');

function ChangeRow({ key, before, after }) {
	const isPrimitive = typeof before !== 'object' && typeof after !== 'object';
	const isSame = before == after
	return html`
  <tr>
	<td style="vertical-align: top; width: 25%; font-weight: bold;">${key}</td>
	<td style="width: 35%; background: #f8f8f8;">
	  ${isPrimitive ? String(before ?? '') : html`<pre>${JSON.stringify(before, null, 2)}</pre>`}
	</td>
	<td style="width: 35%; background: #f8f8f8;">
	  ${isPrimitive ? isSame ? "No changes" : String(after ?? '') : html`<pre>${JSON.stringify(after, null, 2)}</pre>`}
	</td>
  </tr>
`;
}

function Parameters(node) {
	return html`
  <table style="width:100%; border-collapse: collapse; border-spacing:0;">
	<tbody>
	  <tr>
		<td style="vertical-align: top; width: 50%; font-weight: bold;">Address</td>
		<td style="width: 50%; background: #f8f8f8;">${node.address}</td>
	  </tr>
	  <tr>
		<td style="vertical-align: top; width: 50%; font-weight: bold;">Name</td>
		<td style="width: 50%; background: #f8f8f8;">${node.name}</td>
	  </tr>
	  <tr>
		<td style="vertical-align: top; width: 50%; font-weight: bold;">Type</td>
		<td style="width: 50%; background: #f8f8f8;">${node.type}</td>
	  </tr>
	</tbody>
  </table>
  <h3>Parameters:</h3>
  ${node.parameters.length
			? html`<ul>${node.parameters.map(p => html`<li>${p}</li>`)}</ul>`
			: html`<em>No parameters</em>`}
`;
}

function Changes({ before = {}, after = {} }) {
	const keys = new Set([...Object.keys(before), ...Object.keys(after)]);
	return html`
  <table style="width:100%; border-collapse: collapse; border-spacing:0;">
	<thead>
	  <tr style="background:#eee;">
		<th style="width:25%;text-align:left;">Field</th>
		<th style="width:35%;">Current</th>
		<th style="width:35%;">Planned</th>
	  </tr>
	</thead>
	<tbody>
		  ${[...keys].map(k => ({ key: k, before: before[k], after: after[k] }))
			.filter(val => ((val.before !== null) && (val.before !== undefined)) || ((val.after !== null) && (val.after !== undefined)))
			.map(val => ChangeRow(val))}
	</tbody>
  </table>
`;
}

function Popup({ nodeData }) {
	const before = nodeData.changes?.before || {};
	const after = nodeData.changes?.after || {};

	const renderActive = () => {
		switch (activeTab) {
			case 'changes': return Changes({ before, after });
			default: return Parameters(nodeData);
		}
	};

	return html`
  <div id="popup-content">
	<h2>${nodeData.name || nodeData.id}</h2>

	<div id="tabs">
	  ${['parameters', 'changes'].map(tab =>
		html`
		  <button
			class=${activeTab === tab ? 'active' : ''}
			onclick=${() => update(tab)}>
			${tab === 'parameters'
				? 'Parameters'
				: 'Changes'}
		  </button>
		`
	)}
	</div>

	<div id="tab-content">${renderActive()}</div>
	<button id="close" onclick=${closePopup}>Close</button>
  </div>
`;
}

function update(tab) {
	activeTab = tab;
	renderPopup(currentNodeData);
}

let currentNodeData = null;

function renderPopup(nodeData) {
	currentNodeData = nodeData;
	popupEl.classList.remove('hidden');
	render(popupEl, html`${Popup({ nodeData })}`);
}

function closePopup() {
	popupEl.classList.add('hidden');
}

// -------- Cytoscape event ----------
cy.on('tap', 'node', e => {
	const nodeID = e.target.id();
	const nodeData = graph.nodes[nodeID];
	if (nodeData) renderPopup(nodeData);
});

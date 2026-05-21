import { html, render } from 'https://unpkg.com/lighterhtml?module';

const icons = {
	default: '<svg xmlns="http://www.w3.org/2000/svg" xml:space="preserve" viewBox="31.4 0 449.1 512"><path d="m185.9 89.8 139.2 80.8v161.7l-139.2-80.8zm155.4 80.9v161.7l139.2-80.8V89.8zM31.4 0v161.7l139.2 80.8V80.8zm154.5 431.2L325.2 512V350.3L186 269.5v161.7z" style="fill-rule:evenodd;clip-rule:evenodd;fill:#a067da"/></svg>',
}

const cont = document.getElementById("graph");
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

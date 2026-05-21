import { html, render } from 'https://unpkg.com/lighterhtml?module';

const content = document.getElementById("catalog-content");

render(content, html`<p>Catalog coming soon</p>`);

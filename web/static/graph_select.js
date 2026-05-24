const revisions = new SlimSelect({
  select: '#revisions-dropdown',
  events: {
    afterChange: revHandler,
  }
});
const namespaces = new SlimSelect({
  select: '#namespaces-dropdown',
  events: {
    afterChange: nsHandler,
  },
})

function revHandler(option) {
  const rev = option[0].value;
  const ns = namespaces.getSelected()[0];
  if (!ns || !rev) return;
  const path = `/graph/${encodeURIComponent(ns)}/${encodeURIComponent(rev)}`
  htmx.ajax('GET', path, {
    target: '#graph-content',
    swap: 'outerHTML',
  });
}

function nsHandler(option) {
  const ns = option[0].value
  if (!ns || ns === '') return;
  htmx.ajax('GET', `/namespaces/${encodeURIComponent(ns)}/revisions`, {
    target: '#revisions-dropdown'
  }).then(() => {
    setTimeout(() => revisions.open(), 100)
  });
}

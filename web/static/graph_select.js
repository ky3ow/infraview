let ns = null
const revisions = new SlimSelect({
  select: '#revisions-dropdown',
  events: {
    afterChange: (o) => {
      var rev = o[0].value;
      if (!ns || !rev) return;
      var path = `/graph/${encodeURIComponent(ns)}/${encodeURIComponent(rev)}`
      htmx.ajax('GET', path, {
        target: '#graph-content',
        swap: 'outerHTML',
      });
    }
  }
});
const namespaces = new SlimSelect({
  select: '#namespaces-dropdown',
  events: {
    afterChange: (o) => {
      ns = o[0].value
      if (!ns || ns === '') return;
      htmx.ajax('GET', `/namespaces/${encodeURIComponent(ns)}/revisions`, {
        target: '#revisions-dropdown'
      }).then(() => {
        setTimeout(() => revisions.open(), 100)
      });
    },
  },
})

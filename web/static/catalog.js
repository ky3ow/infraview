const rtf = new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' });

function formatRelative(dt) {
	const diffSec = (Date.now() - dt.getTime()) / 1000;
	const absSec = Math.abs(diffSec);

	let value, unit;
	if (absSec < 60) {
		value = -Math.round(diffSec);
		unit = 'second';
	} else if (absSec < 3600) {
		value = -Math.round(diffSec / 60);
		unit = 'minute';
	} else if (absSec < 86400) {
		value = -Math.round(diffSec / 3600);
		unit = 'hour';
	} else if (absSec < 2_592_000) {
		value = -Math.round(diffSec / 86400);
		unit = 'day';
	} else {
		return null; // fall back to absolute date
	}

	return rtf.format(value, unit);
}

function localizeTimes() {
	document.querySelectorAll('time.revision-time').forEach(el => {
		if (el.dataset.localized) return;
		const dt = new Date(el.getAttribute('datetime'));
		el.textContent = formatRelative(dt) ?? dt.toLocaleDateString();
		el.title = dt.toLocaleString();
		el.dataset.localized = 'true';
	});
}

document.addEventListener('DOMContentLoaded', () => localizeTimes());
document.addEventListener('htmx:afterSwap', () => localizeTimes());

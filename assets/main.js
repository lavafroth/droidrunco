const recycle_icon = '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-recycle" viewBox="0 0 16 16"><path d="M9.302 1.256a1.5 1.5 0 0 0-2.604 0l-1.704 2.98a.5.5 0 0 0 .869.497l1.703-2.981a.5.5 0 0 1 .868 0l2.54 4.444-1.256-.337a.5.5 0 1 0-.26.966l2.415.647a.5.5 0 0 0 .613-.353l.647-2.415a.5.5 0 1 0-.966-.259l-.333 1.242-2.532-4.431zM2.973 7.773l-1.255.337a.5.5 0 1 1-.26-.966l2.416-.647a.5.5 0 0 1 .612.353l.647 2.415a.5.5 0 0 1-.966.259l-.333-1.242-2.545 4.454a.5.5 0 0 0 .434.748H5a.5.5 0 0 1 0 1H1.723A1.5 1.5 0 0 1 .421 12.24l2.552-4.467zm10.89 1.463a.5.5 0 1 0-.868.496l1.716 3.004a.5.5 0 0 1-.434.748h-5.57l.647-.646a.5.5 0 1 0-.708-.707l-1.5 1.5a.498.498 0 0 0 0 .707l1.5 1.5a.5.5 0 1 0 .708-.707l-.647-.647h5.57a1.5 1.5 0 0 0 1.302-2.244l-1.716-3.004z"/></svg>';
const trash_icon = '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-trash" viewBox="0 0 16 16"><path d="M5.5 5.5A.5.5 0 0 1 6 6v6a.5.5 0 0 1-1 0V6a.5.5 0 0 1 .5-.5zm2.5 0a.5.5 0 0 1 .5.5v6a.5.5 0 0 1-1 0V6a.5.5 0 0 1 .5-.5zm3 .5a.5.5 0 0 0-1 0v6a.5.5 0 0 0 1 0V6z"/><path fill-rule="evenodd" d="M14.5 3a1 1 0 0 1-1 1H13v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V4h-.5a1 1 0 0 1-1-1V2a1 1 0 0 1 1-1H6a1 1 0 0 1 1-1h2a1 1 0 0 1 1 1h3.5a1 1 0 0 1 1 1v1zM4.118 4 4 4.059V13a1 1 0 0 0 1 1h6a1 1 0 0 0 1-1V4.059L11.882 4H4.118zM2.5 3V2h11v1h-11z"/></svg>';

var isExtended = {};

function search() {
	text = $('.search').val().toLowerCase();
	$.ajax({
		url: "/apps",
		datatype: 'json',
		contentType: 'application/json',
		method: 'POST',
		data: JSON.stringify({
			"query":text
		}),
		success: function (data) {
			var entries = []
			data['apps'].map(function(app) {
			var description =  $("<div />", {
				class: 'description ' + (isExtended[app.pkg] ? '' : 'collapsed collapsed-after'),
				text: app.description,
			});

			var entry = $("<div />", {
				class: 'entry',
				click: function(e){
					parent = e.target.parentElement
					if (
						parent.className == 'action' || parent.tagName == 'svg'
					) {
						$.post("/do", JSON.stringify({
							pkg: app.pkg,
						}));
						search();
						return;
					}
					if (!isExtended[app.pkg]) {
						description.removeClass("collapsed-after", 250);
						description.removeClass("collapsed", 250);
						isExtended[app.pkg] = true;
						return;
					}
					description.addClass("collapsed", 250);
					description.addClass("collapsed-after", 250);
					isExtended[app.pkg] = false;
        			}
			});

			var action = $('<div />', {
				class: 'action',
			});

			action.append($(app.installed ? trash_icon : recycle_icon));
			entry.append(
				action,
				$("<div />", {
					class: 'label',
					text: app.label,
				}),

				$("<div />", {
					class: 'package',
					text: app.pkg,
				}),
				description,
				);
				entries.push(entry);
			});
			$('.container .entry').remove();
			$('.container').append(entries);
		}
	});
}

$('.search').on('input', search);
$(setInterval(search, 2000));
$(search);

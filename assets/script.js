const recycle_icon = '<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" fill="currentColor" class="bi bi-recycle" viewBox="0 0 16 16"><path d="M9.302 1.256a1.5 1.5 0 0 0-2.604 0l-1.704 2.98a.5.5 0 0 0 .869.497l1.703-2.981a.5.5 0 0 1 .868 0l2.54 4.444-1.256-.337a.5.5 0 1 0-.26.966l2.415.647a.5.5 0 0 0 .613-.353l.647-2.415a.5.5 0 1 0-.966-.259l-.333 1.242-2.532-4.431zM2.973 7.773l-1.255.337a.5.5 0 1 1-.26-.966l2.416-.647a.5.5 0 0 1 .612.353l.647 2.415a.5.5 0 0 1-.966.259l-.333-1.242-2.545 4.454a.5.5 0 0 0 .434.748H5a.5.5 0 0 1 0 1H1.723A1.5 1.5 0 0 1 .421 12.24l2.552-4.467zm10.89 1.463a.5.5 0 1 0-.868.496l1.716 3.004a.5.5 0 0 1-.434.748h-5.57l.647-.646a.5.5 0 1 0-.708-.707l-1.5 1.5a.498.498 0 0 0 0 .707l1.5 1.5a.5.5 0 1 0 .708-.707l-.647-.647h5.57a1.5 1.5 0 0 0 1.302-2.244l-1.716-3.004z"/></svg>';
const trash_icon = '<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" preserveAspectRatio="xMidYMid meet" viewBox="0 0 24 24"><path fill="currentColor" fill-rule="evenodd" d="M16 1.75V3h5.25a.75.75 0 0 1 0 1.5H2.75a.75.75 0 0 1 0-1.5H8V1.75C8 .784 8.784 0 9.75 0h4.5C15.216 0 16 .784 16 1.75zm-6.5 0a.25.25 0 0 1 .25-.25h4.5a.25.25 0 0 1 .25.25V3h-5V1.75z"/><path fill="currentColor" d="M4.997 6.178a.75.75 0 1 0-1.493.144L4.916 20.92a1.75 1.75 0 0 0 1.742 1.58h10.684a1.75 1.75 0 0 0 1.742-1.581l1.413-14.597a.75.75 0 0 0-1.494-.144l-1.412 14.596a.25.25 0 0 1-.249.226H6.658a.25.25 0 0 1-.249-.226L4.997 6.178z"/><path fill="currentColor" d="M9.206 7.501a.75.75 0 0 1 .793.705l.5 8.5A.75.75 0 1 1 9 16.794l-.5-8.5a.75.75 0 0 1 .705-.793zm6.293.793A.75.75 0 1 0 14 8.206l-.5 8.5a.75.75 0 0 0 1.498.088l.5-8.5z"/></svg>';

var extended = '';

function accordion(id) {
    extended = extended == id ? '' : id;
}

function search() {
    $.ajax({
        url: "/",
        datatype: 'json',
        contentType: 'application/json',
        method: 'POST',
        data: JSON.stringify({
            "pkg": $('#search').val().toLowerCase()
        }),
        success: function(data) {
            var entries = []
	    if (data == null) {
		return
	    }
            data.map(function(app) {
                var ID = app.pkg.replaceAll('.', '');
                var icon = app.enabled ? trash_icon : recycle_icon;
                var show = ID == extended ? ' show' : '';
                var description = app.description.replaceAll('\n', "<br />");
                var color = 'secondary';
                        if (app.removal == "Recommended") {
                                        color = 'success';
                        } else if (app.removal == "Advanced") {
                                        color = 'primary';
                        } else if (app.removal == "Expert") {
                                color = "warning";
                        } else if (app.removal == "Unsafe") {
                                color = "danger";
                        }
                entries.push(`<div class="accordion-item">
        <h2 class="accordion-header" id="heading${ID}">
                <button class="accordion-button" type="button" data-bs-toggle="collapse" data-bs-target="#collapse${ID}" aria-expanded="true" aria-controls="collapse${ID}" onclick="accordion('${ID}')">
                <h5 class="mb-1">${app.label}
			<span class="badge badge-secondary">${app.list}</span>
		</h5>
                        ${app.pkg}
                </button>
       <button type="button" id="button${ID}" class="btn btn-outline-${color} m-1 rbtn" onclick="toggle('${app.pkg}')">
                ${icon}
                </button>
        </h2>
        <div id="collapse${ID}" class="accordion-collapse collapse${show}" aria-labelledby="heading${ID}" data-bs-parent="#accordion">
          <div class="accordion-body">${description}</div></div>`);
            });
            $('.accordion .accordion-item').remove();
            $('.accordion').append(entries);
        }
    });
}

function toggle(pkg) {
    $.ajax({
        url: "/",
        datatype: 'json',
        method: 'PATCH',
        contentType: 'application/json',
        data: JSON.stringify({
            pkg: pkg,
        })
    });
    search();
}

$('.search').on('input', search);
$(setInterval(search, 1000));
$(search);

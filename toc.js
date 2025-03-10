// Populate the sidebar
//
// This is a script, and not included directly in the page, to control the total size of the book.
// The TOC contains an entry for each page, so if each page includes a copy of the TOC,
// the total size of the page becomes O(n**2).
class MDBookSidebarScrollbox extends HTMLElement {
    constructor() {
        super();
    }
    connectedCallback() {
        this.innerHTML = '<ol class="chapter"><li class="chapter-item expanded affix "><a href="index.html">Accurate</a></li><li class="chapter-item expanded affix "><li class="part-title">User manual</li><li class="chapter-item expanded "><a href="overview.html"><strong aria-hidden="true">1.</strong> Overview</a></li><li class="chapter-item expanded "><a href="concepts.html"><strong aria-hidden="true">2.</strong> Concepts</a></li><li class="chapter-item expanded "><a href="getting_started.html"><strong aria-hidden="true">3.</strong> Getting started</a></li><li><ol class="section"><li class="chapter-item expanded "><a href="config.html"><strong aria-hidden="true">3.1.</strong> Configurations</a></li><li class="chapter-item expanded "><a href="setup.html"><strong aria-hidden="true">3.2.</strong> Deploying Accurate</a></li><li class="chapter-item expanded "><a href="helm.html"><strong aria-hidden="true">3.3.</strong> Helm Chart</a></li><li class="chapter-item expanded "><a href="install-plugin.html"><strong aria-hidden="true">3.4.</strong> Installing kubectl plugin</a></li></ol></li><li class="chapter-item expanded "><a href="usage.html"><strong aria-hidden="true">4.</strong> Usage</a></li><li><ol class="section"><li class="chapter-item expanded "><a href="info.html"><strong aria-hidden="true">4.1.</strong> Showing information</a></li><li class="chapter-item expanded "><a href="templates.html"><strong aria-hidden="true">4.2.</strong> Setting up templates</a></li><li class="chapter-item expanded "><a href="propagation.html"><strong aria-hidden="true">4.3.</strong> Propagating resources</a></li><li class="chapter-item expanded "><a href="subnamespaces.html"><strong aria-hidden="true">4.4.</strong> Sub-namespace operations</a></li></ol></li><li class="chapter-item expanded "><li class="part-title">References</li><li class="chapter-item expanded "><a href="crd_subnamespace.html"><strong aria-hidden="true">5.</strong> SubNamespace custom resource</a></li><li class="chapter-item expanded "><a href="commands.html"><strong aria-hidden="true">6.</strong> Commands</a></li><li><ol class="section"><li class="chapter-item expanded "><a href="kubectl-accurate.html"><strong aria-hidden="true">6.1.</strong> kubectl-accurate</a></li><li class="chapter-item expanded "><a href="accurate-controller.html"><strong aria-hidden="true">6.2.</strong> accurate-controller</a></li></ol></li><li class="chapter-item expanded "><a href="labels.html"><strong aria-hidden="true">7.</strong> Labels</a></li><li class="chapter-item expanded "><a href="annotations.html"><strong aria-hidden="true">8.</strong> Annotations</a></li><li class="chapter-item expanded affix "><li class="part-title">Developer documents</li><li class="chapter-item expanded "><a href="design.html"><strong aria-hidden="true">9.</strong> Design notes</a></li><li class="chapter-item expanded "><a href="reconcile.html"><strong aria-hidden="true">10.</strong> Reconciliation rules</a></li><li class="chapter-item expanded "><a href="release.html"><strong aria-hidden="true">11.</strong> Release procedure</a></li><li class="chapter-item expanded "><a href="maintenance.html"><strong aria-hidden="true">12.</strong> Maintenance</a></li></ol>';
        // Set the current, active page, and reveal it if it's hidden
        let current_page = document.location.href.toString().split("#")[0];
        if (current_page.endsWith("/")) {
            current_page += "index.html";
        }
        var links = Array.prototype.slice.call(this.querySelectorAll("a"));
        var l = links.length;
        for (var i = 0; i < l; ++i) {
            var link = links[i];
            var href = link.getAttribute("href");
            if (href && !href.startsWith("#") && !/^(?:[a-z+]+:)?\/\//.test(href)) {
                link.href = path_to_root + href;
            }
            // The "index" page is supposed to alias the first chapter in the book.
            if (link.href === current_page || (i === 0 && path_to_root === "" && current_page.endsWith("/index.html"))) {
                link.classList.add("active");
                var parent = link.parentElement;
                if (parent && parent.classList.contains("chapter-item")) {
                    parent.classList.add("expanded");
                }
                while (parent) {
                    if (parent.tagName === "LI" && parent.previousElementSibling) {
                        if (parent.previousElementSibling.classList.contains("chapter-item")) {
                            parent.previousElementSibling.classList.add("expanded");
                        }
                    }
                    parent = parent.parentElement;
                }
            }
        }
        // Track and set sidebar scroll position
        this.addEventListener('click', function(e) {
            if (e.target.tagName === 'A') {
                sessionStorage.setItem('sidebar-scroll', this.scrollTop);
            }
        }, { passive: true });
        var sidebarScrollTop = sessionStorage.getItem('sidebar-scroll');
        sessionStorage.removeItem('sidebar-scroll');
        if (sidebarScrollTop) {
            // preserve sidebar scroll position when navigating via links within sidebar
            this.scrollTop = sidebarScrollTop;
        } else {
            // scroll sidebar to current active section when navigating via "next/previous chapter" buttons
            var activeSection = document.querySelector('#sidebar .active');
            if (activeSection) {
                activeSection.scrollIntoView({ block: 'center' });
            }
        }
        // Toggle buttons
        var sidebarAnchorToggles = document.querySelectorAll('#sidebar a.toggle');
        function toggleSection(ev) {
            ev.currentTarget.parentElement.classList.toggle('expanded');
        }
        Array.from(sidebarAnchorToggles).forEach(function (el) {
            el.addEventListener('click', toggleSection);
        });
    }
}
window.customElements.define("mdbook-sidebar-scrollbox", MDBookSidebarScrollbox);

// Expand all and collapse all buttons for folder trees, shared by the read-only and edit views.
// A button with data-expand-all or data-collapse-all acts on every <details> inside the
// nearest element marked data-folder-tree, so several trees can sit on one page.
document.addEventListener("DOMContentLoaded", function() {
    const expandButtons: NodeListOf<HTMLButtonElement> = document.querySelectorAll("button[data-expand-all]");
    const collapseButtons: NodeListOf<HTMLButtonElement> = document.querySelectorAll("button[data-collapse-all]");

    expandButtons.forEach(button => {
        button.addEventListener("click", () => setFoldersOpen(button, true));
    });

    collapseButtons.forEach(button => {
        button.addEventListener("click", () => setFoldersOpen(button, false));
    });
});

// Open or close every folder in the tree containing the button
function setFoldersOpen(button: HTMLButtonElement, open: boolean) {
    const tree: HTMLElement | null = button.closest("[data-folder-tree]");
    if (tree === null) {
        console.error("Button is not inside a folder tree");
        return;
    }

    tree.querySelectorAll("details").forEach(d => {
        d.open = open;
    });
}

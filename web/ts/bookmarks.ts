document.addEventListener("DOMContentLoaded", function() {
    // Add folder buttons
    const folderButtons = document.querySelectorAll("button.add-folder");
    folderButtons.forEach(button => {
        button.addEventListener("click", function(event) {
            if (event === null || event.target === null) {
                console.error(`Failed to add onclick to button: ${button.id}`);
                return;
            }
            const target: HTMLElement = event.target as HTMLElement;
            const parentElement: HTMLElement = target.parentElement as HTMLElement;
            event.preventDefault();
            const parentFolderID: string | undefined = parentElement.dataset.id;
            const name = prompt("Enter the name of the new folder");
            if (name === null) {
                return;
            }

            fetch("/bookmarks/add-folder", {
                method: "POST",
                headers: {
                    "Content-Type": "application/x-www-form-urlencoded"
                },
                body: "parent_folder_id=" + parentFolderID + "&name=" + name
            }).then(function(response) {
                // Check the response status
                if (response.status !== 200) {
                    alert(`Failed to add folder: ${response.statusText}`);
                    return;
                }
                alert(response.status);
                alert(response.headers.get("Content-Type"));
                alert(response.json());
                alert(response.text());
                
                // Parse the JSON response
                response.json().then(function(folder: Folder) {
                    addFolderFromObject(folder);
                });
            });
        });
    });
});

// Define a type for a folder
interface Folder {
    id: number;
    name: string;
    parent_folder_id: number | undefined;
}

// Add a folder from Object
function addFolderFromObject(folder: Folder) {
    // Create folder elements
    const folderTree = document.createElement("ul");
    const folderLi = document.createElement("li");
    const folderDetails = document.createElement("details");
    const folderSummary = document.createElement("summary");
    const folderName = document.createElement("span");
    const folderActions = document.createElement("span");

    // Set foldertree ID and class
    folderTree.id = `folder-${folder.id}`;
    folderTree.classList.add("foldertree");

    // set folder li class
    folderLi.classList.add("folder");

    // Set folder name
    folderName.innerText = folder.name;
    folderName.classList.add("folder-name");

    // Set folder data-id
    folderDetails.dataset.id = folder.id.toString();
    // Set folder actions
    folderActions.innerHTML = `
        <button class="add-bookmark">+ Bookmark</button>
        <button class="add-folder">+ Folder</button>
    `;

    // Set folder summary
    folderSummary.appendChild(folderName);
    folderSummary.appendChild(folderActions);

    // Set folder details
    folderDetails.appendChild(folderSummary);
    folderDetails.appendChild(document.createElement("ul"));
    folderLi.appendChild(folderDetails);
    folderTree.appendChild(folderLi);

    // Add folder to folder tree
    const parentDiv: HTMLElement | null = document.getElementById(`folder-${folder.parent_folder_id}`);
    if (parentDiv === null) {
        console.error(`Failed to find parent folder: ${folder.parent_folder_id}`);
        return;
    }
    parentDiv.appendChild(folderDetails);
}

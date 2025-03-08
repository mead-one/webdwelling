document.addEventListener("DOMContentLoaded", function() {
    const folderButtons = document.querySelectorAll("button.add-folder");
    const addFolderForm = document.getElementById("add-folder-form");
    const cancelAddFolderButton = document.getElementById("cancel-add-folder");

    // Add folder buttons
    folderButtons.forEach(button => {
        button.addEventListener("click", openAddFolderModal);
    });

    addFolderForm.addEventListener("submit", function(event) {
        event.preventDefault();
        const parentFolderID = event.target.parentElement.dataset.id;
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
            
            //window.location.reload();

            // Parse the JSON response
            //const folder = response.json();
            //addFolderFromObject(folder);
        });
    });
    cancelAddFolderButton.addEventListener("click", cancelAddFolderModal);
});

// Open the add folder modal
function openAddFolderModal(event) {
    const parentFolderID = event.target.parentElement.dataset.id;

    const modalContainer = document.getElementById("modal-container");
    const addFolderForm = document.getElementById("add-folder-form");
    const folderSelectWrapper = document.getElementById("folder-select-wrapper");
    const folderSelect = document.getElementById("folder-select");

    folderSelectWrapper.appendChild(folderSelect);
    addFolderForm.reset();
    addFolderForm.elements["folder-select"].value = parentFolderID;
    modalContainer.style.display = "block";
    addFolderForm.style.display = "block";
}

// Close the add folder modal
function cancelAddFolderModal() {
    const modalContainer = document.getElementById("modal-container");
    const addFolderForm = document.getElementById("add-folder-form");

    modalContainer.style.display = "none";
    addFolderForm.style.display = "none";
}

// Add a folder from Object
function addFolderFromObject(folder) {
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
    folderDetails.dataset.id = folder.id;
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
    const parentDiv = document.getElementByID(`folder-${folder.parent_folder_id}`);
    parentDiv.appendChild(folderDetails);
}

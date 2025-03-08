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
        const name = event.target.elements["name"].value;
        const public = event.target.elements["public"].checked;
        const parentFolderID = event.target.elements["folder-select"].value;
        if (name === null) {
            return;
        }

        console.log(`Adding folder ${name} to folder ${parentFolderID}`);

        fetch("/bookmarks/add-folder", {
            method: "POST",
            headers: {
                "Content-Type": "application/x-www-form-urlencoded"
            },
            body: "name=" + name + "&public=" + public + "&parent_folder_id=" + parentFolderID
        }).then(function(response) {
            if (!response.ok) {
                throw new Error(response.statusText);
            }
            return response.json();
        }).then(function(data) {
            // Check the response status
            console.log(data);

            // Add the folder to the folder tree and folder-select-tree
            updateNewFolder(data.name, data.parent_folder_id, data.id);
            closeAddFolderModal();
        }).catch(function(error) {
            alert(`Failed to add folder: ${error}`);
        });
    });
    cancelAddFolderButton.addEventListener("click", closeAddFolderModal);
});

// Update the folder tree and folder select tree after adding a new folder
function updateNewFolder(name, parentFolderID, newID) {
    addFolderToFolderTree(name, parentFolderID, newID);
    addFolderToFolderSelectTree(name, parentFolderID, newID);
}

// Add a folder to the folder tree
function addFolderToFolderTree(name, parentFolderID, newID) {
    const parentFolder = document.getElementById(`folder-${parentFolderID}`);
    const parentDiv = parentFolder.querySelector("details");
    const bookmarksUl = parentFolder.querySelector("ul.bookmarks");
    const folderTree = document.createElement("ul");
    const folderLi = document.createElement("li");
    const folderDetails = document.createElement("details");
    const folderSummary = document.createElement("summary");
    const folderName = document.createElement("span");
    const folderActions = document.createElement("span");
    const addFolderButton = document.createElement("button");
    const folderUl = document.createElement("ul");

    // Set foldertree ID
    folderTree.id = `folder-${newID}`;
    folderTree.classList.add("foldertree");

    // set folder li class
    folderLi.classList.add("folder");

    // Set folder name
    folderName.innerText = `📁 ${name}`;
    folderName.classList.add("folder-name");

    // Set folder data-id
    folderDetails.dataset.id = parentFolderID;
    // Set folder actions
    folderActions.classList.add("folder-actions");
    folderActions.innerHTML = `
        <button class="add-bookmark">+ Bookmark</button>
    `;
    addFolderButton.classList.add("add-folder");
    addFolderButton.innerText = "+ Folder";
    addFolderButton.addEventListener("click", openAddFolderModal);
    folderActions.appendChild(addFolderButton);

    // Set folder ul class
    folderUl.classList.add("bookmarks");

    // Set folder summary
    folderSummary.appendChild(folderName);
    folderSummary.appendChild(folderActions);
    // Set folder details
    folderDetails.appendChild(folderSummary);
    folderDetails.appendChild(document.createElement("ul"));
    folderDetails.appendChild(folderUl);
    folderLi.appendChild(folderDetails);
    folderTree.appendChild(folderDetails);

    // Add folder to folder tree
    parentDiv.appendChild(folderTree);
    // Place the bookmarks ul at the end of the folder tree
    parentDiv.appendChild(bookmarksUl);
}

function addFolderToFolderSelectTree(name, parentFolderID, newID) {
    const parentDiv = document.getElementById(`folder-select-tree-${parentFolderID}`)
    const folderSelect = document.createElement("input");

    folderSelect.type = "radio";
    folderSelect.name = "folder-select";
    folderSelect.value = newID;
    folderSelect.id = `folder-select-${parentFolderID}`;

    const folderSelectLabel = document.createElement("label");
    folderSelectLabel.innerText = `📁 ${name}`;
    folderSelectLabel.for = folderSelect.id;

    const folderSelectLi = document.createElement("li");
    folderSelectLi.classList.add("folder-select");
    folderSelectLi.appendChild(folderSelect);
    folderSelectLi.appendChild(folderSelectLabel);

    parentDiv.appendChild(folderSelectLi);
}

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
function closeAddFolderModal() {
    const modalContainer = document.getElementById("modal-container");
    const addFolderForm = document.getElementById("add-folder-form");

    modalContainer.style.display = "none";
    addFolderForm.style.display = "none";
    addFolderForm.reset();
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

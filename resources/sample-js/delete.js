// Process data before deleting pets
// Available variables: this (data), me (user session), query, isRoot

// Validate permissions - only authenticated users or root can delete
if (!me && !isRoot) {
    cancel('Authentication required to delete pets');
}

// Prevent deletion of adopted pets without special permission
if (this.adoptionStatus === 'adopted' && !isRoot) {
    cancel('Cannot delete adopted pets. Please contact an administrator.');
}

// Log the deletion attempt
console.log('Attempting to delete pet:', this.name, 'by user:', me ? me.id : 'root');

// Business logic - check if pet has important records
if (this.notes && this.notes.includes('medical')) {
    console.log('WARNING: Deleting pet with medical records:', this.name);
}

// Soft delete option - instead of actual deletion, mark as inactive
// Uncomment the following lines to implement soft delete:
// this.status = 'deleted';
// this.deletedAt = new Date().toISOString();
// this.deletedBy = me ? me.id : 'root';
// cancel('Pet marked as deleted instead of permanent deletion');